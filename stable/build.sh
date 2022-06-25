#!/usr/bin/env bash

helm dependency update .
helm dependency build .
