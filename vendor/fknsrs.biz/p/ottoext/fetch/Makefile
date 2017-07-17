dist-fetch.rice-box.go: dist-fetch/bundle.js
	rice embed-go

dist-fetch/bundle.js: js/bundle.js
	mkdir -p dist-fetch
	cp -a js/bundle.js dist-fetch/bundle.js
	cp -a js/bundle.js.map dist-fetch/bundle.js.map

js/bundle.js: js/fetch.js js/headers.js js/index.js js/request.js js/response.js
	cd js && npm install && npm run bundle${BUNDLE_SUFFIX}

clean:
	rm -rf dist-fetch.rice-box.go dist-fetch js/bundle.js
