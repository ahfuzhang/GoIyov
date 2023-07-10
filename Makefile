build:
	go build -o https_proxy cmd/https_proxy_logger/main.go 

run:
	./https_proxy -addr=:8888 -log_file=/dev/stdout

test:
	curl --proxy "http://127.0.0.1:8888" "https://www.sogou.com/" -v -k
	
			