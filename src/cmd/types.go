// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 16:45
// Original filename: src/cmd/types.go

package cmd

import "dtools2/rest"

// Global flags used for option parsing by COBRA

var ConnectURI string
var APIVersion string
var UseTLS bool
var TLSCACert string
var TLSCert string
var TLSKey string
var TLSSkipVerify bool

// Resolved REST client, shared by subcommands.
var restClient *rest.Client

// Auth / login-related flags.

var loginUsername string
var loginPassword string
var loginInsecure bool
var loginCACertPath string

// Image-related flags.

var imagePullRegistry string
