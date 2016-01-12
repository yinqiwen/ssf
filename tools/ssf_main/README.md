# ssf_main
The simplest streaming process application build on ssf fraework.

# Usage
 
    $ bin/ssf_main -h
    Usage of bin/ssf_main:
    -cluster string
        cluster name&servers (default "example/127.0.0.1:48100,127.0.0.1:48101,127.0.0.1:48102")
     -home string
        application home dir (default "./")
     -listen string
        listen addr (default "127.0.0.1:48100")
     -dispatch string
        message dispatch config, format <processor>/<proto name>[,<proto name>][&<processor>/<proto name>[,<proto name>...] (default "wc/ssf.RawMessage,main.Word")
      ....(some glog options)

# Example
## Launch cluster app with 2 static servers and specified message dispatch config
    bin/ssf_main -home ./0 -listen 127.0.0.1:48100 -cluster example/127.0.0.1:48100,127.0.0.1:48101 -dispatch wc/ssf.RawMessage,main.Word  -logtostderr









