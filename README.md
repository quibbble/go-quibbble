# Go-boardgame-networking-internal

## Build and Deploy
```
$ export DOCKER_DEFAULT_PLATFORM=linux/amd64
$ docker build --tag quibbble/quibbble:${TAG} -f build/Dockerfile .
$ docker push quibbble/quibbble:${TAG}
$ docker pull quibbble/quibbble:${TAG}
$ docker run -d --name quibbble -p 8080:8080 quibbble/quibbble:${TAG}
```
