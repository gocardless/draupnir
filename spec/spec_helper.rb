# frozen_string_literal: true
require 'rest_client'
require 'json'
require 'rspec'

# Start the VM if it's not running
RSpec.configure do |r|
  r.before(:example) do
    status = `vagrant status --machine-readable`
    state = status.match(/^\d+,default,state,(\w+)$/)[1]
    `vagrant up` unless state == 'running'
  end
end
