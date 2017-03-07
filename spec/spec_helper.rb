# frozen_string_literal: true
require 'rest_client'
require 'json'
require 'rspec'

JSONAPI_CONTENT_TYPE = "application/vnd.api+json"
SERVER_IP = "192.168.2.3"
SERVER_ADDR = "#{SERVER_IP}:8080"
