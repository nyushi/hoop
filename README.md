Hoop
====

network forwarder

API
---

### /ports/tcp/`localport`

- POST/PUT
	- add forward rule for tcp port `localport`
	- body format: `host:port`
- DELETE
	- delete forward rule

### /ports/udp/`localport`

- POST/PUT
	- add forward rule for udp port `localport`
	- body format: `host:port`
- DELETE
	- delete forward rule

### /ports

- list all ports
