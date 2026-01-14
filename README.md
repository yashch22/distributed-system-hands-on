# Distributed System Hands-On

This repository contains implementations of various distributed systems concepts and algorithms.

## Project: MapReduce

This project implements a distributed MapReduce system in Go, consisting of a coordinator process that manages tasks and multiple worker processes that execute map and reduce operations.

### Implementation Files

- `src/mr/coordinator.go` - Coordinator that assigns tasks to workers and tracks their progress
- `src/mr/worker.go` - Worker that processes map and reduce tasks
- `src/mr/rpc.go` - RPC definitions for communication between coordinator and workers

### Building the Project

Build the MapReduce application plugins. For example, to build the word count application:

```bash
cd src/mrapps
go build -buildmode=plugin wc.go
```

This will create `wc.so` in the `mrapps` directory. You can also build other applications:
- `wc.go` - Word count
- `indexer.go` - Text indexer
- `mtiming.go` - Map timing test
- `rtiming.go` - Reduce timing test
- `jobcount.go` - Job count test
- `early_exit.go` - Early exit test
- `crash.go` - Crash test
- `nocrash.go` - No crash test

### Running MapReduce

1. **Start the coordinator** in one terminal:
   ```bash
   cd src/main
   go run mrcoordinator.go pg-*.txt
   ```
   The coordinator takes input files as arguments. Each file corresponds to one map task.

2. **Start one or more workers** in separate terminals:
   ```bash
   cd src/main
   go run mrworker.go ../mrapps/wc.so
   ```
   You can start multiple workers in parallel by running this command in multiple terminals.

3. **View the results** after completion (from the `main` directory):
   ```bash
   cat mr-out-* | sort | more
   ```
   
   Note: If running in a temporary test directory (like `mr-tmp`), the output files will be there.

### Testing

The project includes a comprehensive test suite. To run all tests:

```bash
cd src/main
bash test-mr.sh
```

This will run the following tests:
- **Word count test** - Basic functionality test
- **Indexer test** - Text indexing application
- **Map parallelism test** - Verifies multiple map tasks run in parallel
- **Reduce parallelism test** - Verifies multiple reduce tasks run in parallel
- **Job count test** - Verifies correct number of jobs executed
- **Early exit test** - Ensures workers don't exit before completion
- **Crash test** - Tests fault tolerance with worker crashes

For quiet output (less verbose):
```bash
bash test-mr.sh quiet
```

You can also run the test suite multiple times to stress test:
```bash
bash test-mr-many.sh 10
```
This will run the test suite 10 times (replace 10 with any number).

### Manual Testing Example

1. Clean previous outputs:
   ```bash
   cd src/main
   rm -rf mr-tmp
   rm -f mr-out*
   ```

2. Build the word count plugin:
   ```bash
   cd ../mrapps
   go build -buildmode=plugin wc.go
   cd ../main
   ```

3. Run the coordinator:
   ```bash
   go run mrcoordinator.go pg-*.txt &
   ```

4. Run multiple workers (in separate terminals):
   ```bash
   go run mrworker.go ../mrapps/wc.so
   ```

5. Compare output with sequential version:
   ```bash
   # Run sequential version for comparison
   go run mrsequential.go ../mrapps/wc.so pg-*.txt
   sort mr-out-0 > mr-correct-wc.txt
   
   # Sort distributed output
   sort mr-out-* | grep . > mr-wc-all
   
   # Compare
   cmp mr-wc-all mr-correct-wc.txt
   ```

### Architecture

- **Coordinator**: Manages task distribution, tracks task states (idle, in-progress, completed), handles timeouts (10 seconds), and ensures all map tasks complete before starting reduce tasks.
- **Worker**: Continuously requests tasks from coordinator, executes map or reduce operations, writes results atomically using temporary files, and reports task completion.
- **Communication**: Uses Unix domain sockets for RPC communication between coordinator and workers.

### Features

- Fault tolerance: Tasks are reassigned if workers crash or timeout
- Atomic writes: Temporary files are used and atomically renamed to prevent partial writes
- Parallel execution: Multiple workers can process tasks concurrently
- Task tracking: Coordinator tracks task states and ensures all tasks complete

## Project: Concurrency Examples

This project contains educational examples demonstrating different concurrency patterns in Go.

### Implementation Files

- `src/concurrency/crawler.go` - Web crawler implementations demonstrating serial, mutex-based, and channel-based concurrency
- `src/concurrency/kv.go` - Key-value store with RPC client-server architecture

### Web Crawler (crawler.go)

Demonstrates three different approaches to implementing a web crawler:

1. **Serial crawler** - Sequential recursive depth-first traversal
2. **Concurrent crawler with mutex** - Uses shared state protected by mutex synchronization
3. **Concurrent crawler with channels** - Uses channel-based coordination pattern

Running the crawler:

```bash
cd src/concurrency
go run crawler.go
```

This will run all three implementations and demonstrate their different approaches to web crawling.

### Key-Value Store (kv.go)

Implements a simple distributed key-value store using RPC:

- **Server**: Thread-safe key-value store with Get and Put operations
- **Client**: RPC client that connects to the server and performs operations
- **Communication**: TCP-based RPC on port 1234

Running the key-value store:

```bash
cd src/concurrency
go run kv.go
```

The program starts a server and then performs sample get/put operations.

### Features

- **Multiple concurrency patterns**: Demonstrates mutex-based vs channel-based concurrency
- **RPC communication**: Shows how to implement RPC client-server architecture
- **Thread-safe operations**: Mutex-protected shared state for concurrent access
- **Educational examples**: Reference implementations for common concurrency patterns

