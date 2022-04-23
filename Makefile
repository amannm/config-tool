clean:
	./make.sh clean
prepare: clean
	./make.sh prepare
test: prepare
	./make.sh test
build: test
	./make.sh build
dist: test
	./make.sh dist darwin arm64
	./make.sh dist darwin amd64