package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	ExitTask
	NoTask
)

type GetTaskArgs struct {
	// Empty - worker just requests a task
}

type GetTaskReply struct {
	TaskType  TaskType
	TaskId    int
	FileName  string   // For map tasks
	FileNames []string // For reduce tasks
	NReduce   int
	MapTaskId int // For reduce tasks, to know which map outputs to read
}

type ReportTaskArgs struct {
	TaskType TaskType
	TaskId   int
	Success  bool
}

type ReportTaskReply struct {
	// Empty - just acknowledge
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
