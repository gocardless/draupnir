# frozen_string_literal: true
require "rest_client"

RSpec.describe "/health_check" do
  it "responds with 'OK'" do
    `vagrant up`
    response = RestClient.get("localhost:8080/health_check")
    expect(response.code).to eq(200)
    expect(response.body).to eq("OK")
  end
end
