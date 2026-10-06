# Changelog

## [0.1.2](https://github.com/SpechtLabs/go-gin-prometheus/compare/v0.1.1...v0.1.2) (2026-10-05)


### Bug Fixes

* **deps:** drop controller-runtime and the k8s.io modules from the module graph ([87fbdcb](https://github.com/SpechtLabs/go-gin-prometheus/commit/87fbdcb4a2df8733e8f0c884597fce8bb31a62ae))
* **deps:** require Go 1.27.1 and update all dependencies ([87fbdcb](https://github.com/SpechtLabs/go-gin-prometheus/commit/87fbdcb4a2df8733e8f0c884597fce8bb31a62ae))
* stop the push gateway loop from panicking when the metrics URL is unreachable ([87fbdcb](https://github.com/SpechtLabs/go-gin-prometheus/commit/87fbdcb4a2df8733e8f0c884597fce8bb31a62ae))
* time out push gateway requests after 10 seconds and close their responses ([87fbdcb](https://github.com/SpechtLabs/go-gin-prometheus/commit/87fbdcb4a2df8733e8f0c884597fce8bb31a62ae))
