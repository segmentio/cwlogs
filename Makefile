version := $$CIRCLE_TAG

release: gh-release govendor clean dist
	github-release release \
	--security-token $$GH_LOGIN \
	--user segmentio \
	--repo cwlogs \
	--tag $(version) \
	--name $(version)

	github-release upload \
	--security-token $$GH_LOGIN \
	--user segmentio \
	--repo cwlogs \
	--tag $(version) \
	--name cwlogs-$(version)-darwin-amd64 \
	--file dist/cwlogs-$(version)-darwin-amd64

	github-release upload \
	--security-token $$GH_LOGIN \
	--user segmentio \
	--repo cwlogs \
	--tag $(version) \
	--name cwlogs-$(version)-linux-amd64 \
	--file dist/cwlogs-$(version)-linux-amd64

clean:
	rm -rf ./dist

dist:
	mkdir dist
	govendor sync
	GOOS=darwin GOARCH=amd64 go build -o dist/cwlogs-$(version)-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -o dist/cwlogs-$(version)-linux-amd64

gh-release:
	go get -u github.com/aktau/github-release

govendor:
	go get -u github.com/kardianos/govendor
