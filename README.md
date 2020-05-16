# gomcli

*gomcli* is a simple Go library to help developers build command-line interfaces in the style of [Metasploit Framework](https://www.metasploit.com/)'s *msfconsole*.

It is heavily inspired by [riposte](https://github.com/fwkz/riposte), a similar library for Python (in fact, some chunks of the code are a direct conversion to Go). However, the scope is a bit narrower, since things like output formatting are intentionally left out of the library's functionality.

## Usage

## Dependencies

* [Liner](https://github.com/peterh/liner)
* [go-shlex](github.com/anmitsu/go-shlex)