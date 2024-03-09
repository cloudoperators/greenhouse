.PHONY: content
content:
	scripts/copy-content-from-docs

.PHONY: clean
clean:
	rm -rf _vendor content/docs/**

.PHONY: serve
serve: clean content
	docker run -p 1313:1313 --rm -v $(CURDIR):/src klakegg/hugo:0.111.3-ext-debian server --disableFastRender
