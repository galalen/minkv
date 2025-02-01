package minkv

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// timestamp (4 bytes) + key len (4 bytes) + value len (4 bytes) + tombstone (1 byte)
const headerSize = 13
const tombstoneOffset = -1

type Record struct {
	Key       []byte
	Value     []byte
	Timestamp uint32
	Tombstone bool
}

type Store struct {
	filename string
	mu       sync.RWMutex
	file     *os.File
	index    map[string]int64
}

func Open(filename string) (*Store, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	store := &Store{
		filename: filename,
		file:     file,
		index:    make(map[string]int64),
	}

	if err := store.buildIndex(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to rebuild index: %w", err)
	}

	return store, nil
}

func (s *Store) buildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var offset int64
	for {
		record, err := s.readRecord(offset)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read record: %v", err)
		}

		// mark tombstoned records as deleted
		if record.Tombstone {
			s.index[string(record.Key)] = tombstoneOffset
		} else {
			s.index[string(record.Key)] = offset
		}
		offset += int64(headerSize + len(record.Key) + len(record.Value))
	}
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *Store) writeRecord(record *Record) (int64, error) {
	totalSize := headerSize + len(record.Key) + len(record.Value)
	buf := make([]byte, totalSize)

	// serialize timestamp (4 bytes)
	binary.BigEndian.PutUint32(buf[0:4], record.Timestamp)

	// serialize key len (4 bytes)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(record.Key)))

	// serialize value len (4 bytes)
	binary.BigEndian.PutUint32(buf[8:12], uint32(len(record.Value)))

	// serialize tombstone (1 byte)
	if record.Tombstone {
		buf[12] = 1
	} else {
		buf[12] = 0
	}

	// copy key data
	copy(buf[headerSize:headerSize+len(record.Key)], record.Key)

	// copy value data
	copy(buf[headerSize+len(record.Key):], record.Value)

	// write record to file
	offset, err := s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	if _, err := s.file.Write(buf); err != nil {
		return 0, err
	}

	return offset, nil
}

func (s *Store) Put(key, value []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := &Record{
		Key:       key,
		Value:     value,
		Timestamp: uint32(time.Now().Unix()),
	}

	offset, err := s.writeRecord(record)
	if err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	s.index[string(key)] = offset
	return nil
}

func (s *Store) readRecord(offset int64) (*Record, error) {
	if _, err := s.file.Seek(offset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	record := &Record{}

	// read timestamp (4 bytes)
	timestampBuf := make([]byte, 4)
	if _, err := s.file.Read(timestampBuf); err != nil {
		return nil, err
	}
	record.Timestamp = binary.BigEndian.Uint32(timestampBuf)

	// read key len (4 bytes)
	keyLenBuf := make([]byte, 4)
	if _, err := s.file.Read(keyLenBuf); err != nil {
		return nil, err
	}
	keyLen := binary.BigEndian.Uint32(keyLenBuf)

	// read value len (4 bytes)
	valueLenBuf := make([]byte, 4)
	if _, err := s.file.Read(valueLenBuf); err != nil {
		return nil, err
	}
	valueLen := binary.BigEndian.Uint32(valueLenBuf)

	// read tombstone (1 byte)
	tombstoneBuf := make([]byte, 1)
	if _, err := s.file.Read(tombstoneBuf); err != nil {
		return nil, err
	}
	record.Tombstone = tombstoneBuf[0] == 1

	// read key data
	key := make([]byte, keyLen)
	if _, err := s.file.Read(key); err != nil {
		return nil, err
	}
	record.Key = key

	// read value data
	value := make([]byte, valueLen)
	if _, err := s.file.Read(value); err != nil {
		return nil, err
	}
	record.Value = value

	return record, nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	offset, exists := s.index[string(key)]
	if !exists || offset == tombstoneOffset {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	record, err := s.readRecord(offset)
	if err != nil {
		return nil, fmt.Errorf("failed to read record: %w", err)
	}

	return record.Value, nil
}

func (s *Store) Delete(key []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("key cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	record := &Record{
		Key:       key,
		Tombstone: true,
		Timestamp: uint32(time.Now().Unix()),
	}

	if _, err := s.writeRecord(record); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	s.index[string(key)] = tombstoneOffset
	return nil
}

type Iterator interface {
	Next() bool
	Record() (*Record, error)
}

type storeIterator struct {
	store    *Store
	record   *Record
	offset   int64
	fileSize int64
	err      error
}

func (s *Store) Iterator() (Iterator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fileInfo, err := s.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &storeIterator{
		store:    s,
		offset:   0,
		fileSize: fileInfo.Size(),
	}, nil
}

func (it *storeIterator) Next() bool {
	for it.offset < it.fileSize {
		record, err := it.store.readRecord(it.offset)
		if err != nil {
			it.err = fmt.Errorf("failed to read record at offset %d: %w", it.offset, err)
			return false
		}

		recordSize := headerSize + len(record.Key) + len(record.Value)

		// skip tombstoned record
		if record.Tombstone {
			it.offset += int64(recordSize)
			continue
		}

		off, exists := it.store.index[string(record.Key)]
		if exists && off == it.offset {
			it.record = record
			it.offset += int64(recordSize)
			return true
		}

		it.offset += int64(recordSize)
	}

	it.record = nil
	return false
}

func (it *storeIterator) Record() (*Record, error) {
	if it.err != nil {
		return nil, it.err
	}

	return it.record, nil
}
