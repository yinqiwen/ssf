# Example
Simple word count example of how to use ssf to build a simple distribute stream processing system.

# Setup
Before test, we need build 3 apps first.

- ssf_main, a framework process which do all transport/cluster work.
- wc, the process which do the word count task.
- textevent, an app to generate text event from stdin and send them to specified servers.

Follow the steps below to build the 3 apps.

    $ go build github.com/yinqiwen/ssf/example/ssf_main
    $ go build github.com/yinqiwen/ssf/example/wc
    $ go build github.com/yinqiwen/ssf/tools/textevent       
    

# Test
In this example we would launch 2 framework process in same machine to simulate a cluster with 2 nodes. The number of cluster nodes could be modified by 'ssf_main' argument '-cluster'.

Before all test, assume that
## Launch SSF Framework
The example ssf framework process usage:
    
    $ bin/ssf_main -h
    Usage of bin/ssf_main:
    -cluster string
        cluster name&servers (default "example/127.0.0.1:48100,127.0.0.1:48101,127.0.0.1:48102")
     -home string
        application home dir (default "./")
     -listen string
        listen addr (default "127.0.0.1:48100")
      ....(some glog options)

It's better to launch the 2 process in 2 different terminal consoles to watch the output.

    $ bin/ssf_main -cluster example/127.0.0.1:48100,127.0.0.1:48101 -listen 127.0.0.1:48100 -home ./node0 -logtostderr
    $ bin/ssf_main -cluster example/127.0.0.1:48100,127.0.0.1:48101 -listen 127.0.0.1:48101 -home ./node1 -logtostderr

## Launch Word Count Processor
Launch word count processor for each SSF framework process.   
It's better to launch the 2 process in 2 different terminal consoles to watch the output.

    $ bin/wc -home ./node0 -cluster example -logtostderr
    $ bin/wc -home ./node1 -cluster example -logtostderr

## Send Text Event To Running System
    $ cat <text file> | bin/textevent -cluster 127.0.0.1:48100
   
## Collect & Print Result
Just send SIGINT to each wc process, the wc process would print result when  SIGINT captured.  
If the wc is running at the foreground, just press Ctrl+C to stop it.







