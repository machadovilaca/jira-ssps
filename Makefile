build:
	go build -ldflags="-s -w" -o _out/jira-ssps ./cmd

run: build
	./_out/jira-ssps
