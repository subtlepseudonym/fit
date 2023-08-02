# Changelog

## [0.3.0] - 2023-08-01
### Added
- Postgres table definition for import run information
- Foreign keys on activity records referencing import table

### Changed
- Silently skip bucket creation if bucket exists in 'etl setup'
- Return "unknown" for type when basic fit file type is unknown

### Fixed
- Reading connection strings from flags in 'etl setup'
- Nil pointer dereference when ETLing activities without sport data

## [0.2.1] - 2023-04-03
### Added
- Tag line records and summary if file checksum is ignored

### Fixed
- Line protocol requires tags to be added in lexical order

## [0.2.0] - 2023-04-03
### Added
- Command 'summarize' for calculating record aggregates
- Command 'etl' for writing summary to postgres and records to influx
- Command 'etl setup' for setting up expected databases / schema
- Command 'inspect' for inspecting populated activity record fields
- Add flag for ignoring file checksum check

### Changed
- Reorganized repo into an importable package
	- This has potential to be confusing in conjuction with subtlepseudonym/fit-go

### Fixed
- Makefile bug for generating automatic versioning
- Fixed panic when activity doesn't container sport information

## [0.1.0] - 2022-10-04
### Added
- Command 'dump' for dumping file information
- Command 'line' for generating line protocol files
- Command 'type' for reading mapped file type information
- Version information
