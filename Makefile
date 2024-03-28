OS :=linux darwin windows
ARCH :=amd64 arm64
PLATFORM_MATRIX :=$(foreach os,$(OS),$(foreach arch,$(ARCH),$(os)_$(arch)))
JOBS := 1

.PHONY: build

release: 
	$(MAKE) clean
	$(MAKE) -j $(JOBS) build
	$(MAKE) -j $(JOBS) pack
	$(MAKE) create-release

build: build-restc build-gin-plugin

clean:
	rm -rf build

build-restc: $(foreach platform,$(PLATFORM_MATRIX),build-restc-$(platform))

build-gin-plugin: $(foreach platform,$(PLATFORM_MATRIX),build-plugin-gin_$(platform))

build-restc-%:
	export GOOS=$(word 1,$(subst _, ,$*)) GOARCH=$(word 2,$(subst _, ,$*)); \
	go build \
	-o build/bin/restc_`echo $$GOOS`_`echo $$GOARCH`/restc \
	cmd/restc.go;

build-plugin-%:
	export PLUGIN=$(word 1,$(subst _, ,$*)) GOOS=$(word 2,$(subst _, ,$*)) GOARCH=$(word 3,$(subst _, ,$*)); \
	go build \
	-o build/bin/restc-`echo $$PLUGIN`_`echo $$GOOS`_`echo $$GOARCH`/restc-`echo $$PLUGIN` \
	plugins/`echo $$PLUGIN`/cmd/restc-`echo $$PLUGIN`.go;

pack: $(foreach file,$(wildcard build/bin/*),pack-$(subst build/bin/,,$(file)))

pack-%:
	tar -czvf build/$*.tar.gz -C build/bin/$* ./

create-release:
	gh release create --generate-notes $(TAG) $(wildcard build/*.tar.gz)