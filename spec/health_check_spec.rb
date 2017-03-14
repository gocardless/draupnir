# frozen_string_literal: true
require 'spec_helper'

RSpec.describe '/health_check' do
  it "responds with 'OK'" do
    response = get("/health_check")
    expect(response.code).to eq(200)
    expect(response.body).to eq("OK\n")
  end
end
