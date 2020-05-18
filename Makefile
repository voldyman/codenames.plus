all:
	pkger -include /server -include /public
	go build
