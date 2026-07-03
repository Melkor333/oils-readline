build:
	podman run -v ./:/code --rm -it alpine:edge sh -c 'apk add bash go g++ gcc wget && cd /code && go generate ./...'
	

