# openbazaar-go
![banner](https://i.imgur.com/iOnXDXK.png)
OpenBazaar Server Daemon in Go

[![Build Status](https://travis-ci.org/OpenBazaar/openbazaar-go.svg?branch=master)](https://travis-ci.org/OpenBazaar/openbazaar-go)
[![Coverage Status](https://coveralls.io/repos/github/OpenBazaar/openbazaar-go/badge.svg?branch=master)](https://coveralls.io/github/OpenBazaar/openbazaar-go?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/OpenBazaar/openbazaar-go)](https://goreportcard.com/report/github.com/OpenBazaar/openbazaar-go)

This repository contains the OpenBazaar server daemon which handles the heavy lifting for the [OpenBazaar](https://openbazaar.org/) desktop application. The server combines several technologies: A modified [IPFS](https://ipfs.io) node, which itself combines ideas from Git, BitTorrent, and Kademlia. A lightweight wallet for interacting with several cryptocurrency networks. And a JSON API which can be used by a user interface to control the node and browse the network. Find the user interface for the server at [github.com/OpenBazaar/openbazaar-desktop](https://github.com/OpenBazaar/openbazaar-desktop).

The Mobazha project is forked from Haven. The Mobazha team would maintain the project for the community.