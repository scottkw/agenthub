#!/bin/bash

wails build -tags wailsassets && xattr -dr com.apple.quarantine build/bin/agenthub.app && open build/bin/agenthub.app
