# frozen_string_literal: true
require 'spec_helper'

RSpec.describe "/base_images" do
  it "responds with 'OK'" do
    response = RestClient.get("localhost:8080/base_images")
    expect(response.code).to eq(200)
    expect(JSON.parse(response.body)).to eq([])
  end
end
