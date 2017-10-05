# frozen_string_literal: true
require 'spec_helper'

RSpec.describe '/health_check' do
  it "responds with 'OK'" do
    response = get("/health_check")
    expect(response.code).to eq(200)
    expect(JSON.parse(response.body)).to eq("status" => "ok")
    expect(response.headers[:content_type]).to eq(JSON_CONTENT_TYPE)
    expect(response.headers[:draupnir_version]).to eq("1.0.0")
  end
end
