# textevent
A command-line tool which could generate text event from STDIN and send them to specified SSF servers.

# Usage

    Usage of textevent:
       -cluster string
            cluster servers (default "127.0.0.1:48100,127.0.0.1:48101,127.0.0.1:48102")


# Example
Generate&send text event to servers 127.0.0.1:48100&127.0.0.1:48101.

    $ cat <text files> | bin/textevent -cluster 127.0.0.1:48100,127.0.0.1:48101
   








