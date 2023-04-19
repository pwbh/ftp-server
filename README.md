# FTP Server

<p align="center">
  <img src="https://user-images.githubusercontent.com/127856937/232848518-e9205774-2839-4491-8d18-0c0ce730e5f2.png" alt="Terminal output example"/>
</p>

An FTP server spec (partial) implementation that I wrote practicing networking programming.

You can read more about the FTP specification in [RFC 959](https://www.rfc-editor.org/rfc/rfc959).

## Usage

Make sure that the FTP client is set to binary mode before transfering files not to get corrupt results.

Usage is pretty standard a file transfer can be inititaed with the `put local/path remote/path` command, do note that the file name and location is hardcoded in the code so server will ignore your `remote/path` (At least for now).

I might add more functionality to it, I might leave it in this state as this was to grasp the core functionality of TCP/IP while also learning Golang.
