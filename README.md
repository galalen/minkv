# MinKV: minimalist key-value store

MinKV is a lightweight, append-only key-value store written in Go, inspired by the Bitcask model. It is designed for simplicity, durability, and efficiency, making it a great learning tool for understanding the inner workings of storage engines.

## Features

- **Append-Only Write Mechanism**: Ensures simplicity and write performance.
- **In-Memory Index**: Enables fast lookups for keys.
- **Thread-Safe Operations**: Ensures concurrent access without data corruption.
- **Support for Basic CRUD Operations**:
    - `Put`: Add or update key-value pairs.
    - `Get`: Retrieve values by key.
    - `Delete`: Remove record.
- **Iterator Support**: Allows scanning through all records.

## Getting Started

### Prerequisites
- Go 1.19 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/galalen/minkv.git
   cd minkv
   ```
2. Running tests:
    ```bash
    go test -v
    ```
   
3. Build the project:
    ```bash
    go build -o minkv
    ```

### Usage

```go
package main

import (
	"fmt"
	
	"github.com/galalen/minkv"
)

func main() {
    store, _ := minkv.Open("store.db")
    defer store.Close()
    
    // Put data
    _ = store.Put([]byte("os"), []byte("mac"))
    _ = store.Put([]byte("db"), []byte("kv"))
    _ = store.Put([]byte("lang"), []byte("go"))
    
    // Retrieve data
    value, _ := store.Get([]byte("os"))
    fmt.Printf("os => %s\n", value)
    
    // update value
    _ = store.Put([]byte("os"), []byte("linux"))
    
    // Delete data
    _ = store.Delete([]byte("db"))
    
    // Iterate over records
    iter, _ := store.Iterator()
    for iter.Next() {
        record, _ := iter.Record()
        fmt.Printf("Key: %s, Value: %s\n", record.Key, record.Value)
    }
}
```

### Future Improvements
- Compaction: Periodically clean up old and deleted keys to reclaim space.
- Hintfiles: Improve read performance by reducing the number of disk seeks.
- Batching Writes: Group multiple writes into a single batch to reduce the number of disk sync operations.

### Contributing
Contributions are welcome! Feel free to fork the repository and submit a pull request.

### Acknowledgments
Inspired by the design of **[py-caskdb](https://github.com/avinassh/py-caskdb)** | **[bitcask](https://github.com/basho/bitcask)** | **[tinykv](https://github.com/talent-plan/tinykv)**

Built with ❤️ and Go.
