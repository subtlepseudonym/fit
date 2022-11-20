# Changelog

## [0.2.0] -
### Added
- Command 'summarize' for calculating record aggregates
- Command 'etl' for writing summary to postgres and records to influx

### Changed
- Reorganized repo into an importable package
	- This has potential to be confusing in conjuction with subtlepseudonym/fit-go

### Fixed
- Makefile bug for generating automatic versioning

## [0.1.0] - 2022-10-04
### Added
- Command 'dump' for dumping file information
- Command 'line' for generating line protocol files
- Command 'type' for reading mapped file type information
- Version information