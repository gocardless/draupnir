# frozen_string_literal: true

require "spec_helper"

RSpec.describe "/health_check" do
  it "responds with 'OK'" do
    response = get("/health_check")
    expect(response.code).to eq(200)
    expect(JSON.parse(response.body)).to eq("status" => "ok")
    expect(response.headers[:content_type]).to eq("application/json")
    expect(response.headers[:draupnir_version]).to eq(Draupnir::VERSION)

    # Verify that we have a well-formed version, ie. valid semver value
    expect(response.headers[:draupnir_version]).to_not eql("0.0.0")
    expect(response.headers[:draupnir_version]).to match(/^\d+\.\d+\.\d+$/)
  end

  context "with old client version" do
    it "responds with 'Bad Request'" do
      expect { client.request(:get, "/images", nil, draupnir_version: "0.0.0") }.
        to raise_error(RestClient::BadRequest) { |err| expect(err.http_code).to be(400) }
    end
  end
end
