#ornet

cmdset is a remote command manager.


ngxfmd: nginx file manager server.


ngxfmd description:
configuration example:
[ngxfmd]
error_log = true
access_log=true
fastcgi_listen_addr = ":11000"
http_listen_addr = ":11001"

[files]
store_path = "/data/store"
upload_type = 2
request_pool_size = 1000

[sandbox]
lua_filename = "/root/myopensrc/ornet/anyd/src/service/sandbox/examples/test.lua"

ngxfmd default support fastcgi and http interfaces, so fastcgi_listen_addr and http_listen_addr
should be configured;

files module used to download and upload files, store_path speicify upload store path, and upload_type means upload type, 1 means upload directly, 2 means use multi-part form way, request_pool_size
means the max concurrent http request at the same time.

sandbox module is used to support lua module to process http request, reference the blog:http://my.oschina.net/shawnChen/blog/380061


more questions? , please mail to cxwshawn@yeah.net;

