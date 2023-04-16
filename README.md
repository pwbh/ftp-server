# FTP Server

An FTP server spec implementation that I wrote (which is pretty broken and probably only I can use it) for practicing networking programming.

You can read more about the FTP specification in [RFC 959](https://www.rfc-editor.org/rfc/rfc959).

## Usage

Make sure that the FTP client is set to binary mode before transfering files not to get corrupt results.

Usage is pretty standard a file transfer can be inititaed with the `put local/path remote/path` command, do note that the file name and location is hardcoded in the code so that server ignores your `remote/path` specified (At least for now).

I might add more functionality to it, I might leave it in this state as this was to grasp the core functionality of TCP/IP while also learning Golang.
