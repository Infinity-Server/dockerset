### VNC password util

###### COMPILE

> Just run make or gcc -o vncpwd vncpwd.c d3des.c

###### USAGE
> vncpwd <vnc password file> <mode> [passwod]

##### EXAMPLE
```
$ vncpwd .vnc/passwd get
$ vncpwd .vnc/passwd set 123
```
