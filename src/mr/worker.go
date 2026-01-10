package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		args := GetTaskArgs{}
		reply := GetTaskReply{}

		ok := call("Coordinator.GetTask", &args, &reply)
		if !ok {
			// Coordinator has exited or is unreachable, worker should exit
			break
		}

		switch reply.TaskType {
		case MapTask:
			success := performMapTask(reply.FileName, reply.TaskId, reply.NReduce, mapf)
			reportTask(MapTask, reply.TaskId, success)
		case ReduceTask:
			success := performReduceTask(reply.TaskId, reply.FileNames, reducef)
			reportTask(ReduceTask, reply.TaskId, success)
		case ExitTask:
			// Coordinator says all work is done
			return
		case NoTask:
			// No task available, wait a bit before asking again
			time.Sleep(time.Second)
		}
	}
}

func performMapTask(filename string, mapTaskId int, nReduce int, mapf func(string, string) []KeyValue) bool {
	// Read input file
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Cannot open file %v: %v", filename, err)
		return false
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Cannot read file %v: %v", filename, err)
		file.Close()
		return false
	}
	file.Close()

	// Call Map function
	kva := mapf(filename, string(content))

	// Partition intermediate key/value pairs by reduce task
	buckets := make([][]KeyValue, nReduce)
	for _, kv := range kva {
		reduceId := ihash(kv.Key) % nReduce
		buckets[reduceId] = append(buckets[reduceId], kv)
	}

	// Write intermediate files using temporary files and atomic rename
	intermediateFiles := []*os.File{}
	tempFiles := []string{}
	for reduceId := 0; reduceId < nReduce; reduceId++ {
		// Create temporary file
		tempFile, err := ioutil.TempFile("", fmt.Sprintf("mr-%d-%d-", mapTaskId, reduceId))
		if err != nil {
			log.Printf("Cannot create temp file: %v", err)
			// Clean up already created files
			for _, f := range intermediateFiles {
				f.Close()
				os.Remove(f.Name())
			}
			return false
		}
		tempFiles = append(tempFiles, tempFile.Name())
		intermediateFiles = append(intermediateFiles, tempFile)
	}

	// Write key/value pairs to intermediate files
	for reduceId := 0; reduceId < nReduce; reduceId++ {
		enc := json.NewEncoder(intermediateFiles[reduceId])
		for _, kv := range buckets[reduceId] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Printf("Cannot encode key/value: %v", err)
				// Clean up
				for _, f := range intermediateFiles {
					f.Close()
				}
				for _, tempName := range tempFiles {
					os.Remove(tempName)
				}
				return false
			}
		}
		intermediateFiles[reduceId].Close()
	}

	// Atomically rename temporary files to final intermediate files
	for reduceId := 0; reduceId < nReduce; reduceId++ {
		finalName := fmt.Sprintf("mr-%d-%d", mapTaskId, reduceId)
		err := os.Rename(tempFiles[reduceId], finalName)
		if err != nil {
			log.Printf("Cannot rename temp file to %v: %v", finalName, err)
			// Clean up remaining temp files
			for i := reduceId; i < nReduce; i++ {
				if i < len(tempFiles) {
					os.Remove(tempFiles[i])
				}
			}
			// Clean up already renamed files
			for i := 0; i < reduceId; i++ {
				os.Remove(fmt.Sprintf("mr-%d-%d", mapTaskId, i))
			}
			return false
		}
	}

	return true
}

func performReduceTask(reduceTaskId int, fileNames []string, reducef func(string, []string) string) bool {
	// Read all intermediate files for this reduce task
	kva := []KeyValue{}
	for _, filename := range fileNames {
		file, err := os.Open(filename)
		if err != nil {
			// File might not exist if map task hasn't completed or failed
			// This is okay, we'll just skip it
			continue
		}

		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
		file.Close()
	}

	// Sort by key
	sort.Sort(ByKey(kva))

	// Create temporary output file
	tempFile, err := ioutil.TempFile("", fmt.Sprintf("mr-out-%d-", reduceTaskId))
	if err != nil {
		log.Printf("Cannot create temp output file: %v", err)
		return false
	}
	tempName := tempFile.Name()
	defer tempFile.Close()

	// Call Reduce on each distinct key
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)

		// Write output in the correct format
		fmt.Fprintf(tempFile, "%v %v\n", kva[i].Key, output)

		i = j
	}
	tempFile.Close()

	// Atomically rename to final output file
	outputName := fmt.Sprintf("mr-out-%d", reduceTaskId)
	err = os.Rename(tempName, outputName)
	if err != nil {
		log.Printf("Cannot rename output file to %v: %v", outputName, err)
		os.Remove(tempName)
		return false
	}

	return true
}

func reportTask(taskType TaskType, taskId int, success bool) {
	args := ReportTaskArgs{
		TaskType: taskType,
		TaskId:   taskId,
		Success:  success,
	}
	reply := ReportTaskReply{}
	call("Coordinator.ReportTask", &args, &reply)
}

// For sorting by key
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		// Coordinator may have exited, this is okay
		return false
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
