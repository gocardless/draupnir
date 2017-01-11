# frozen_string_literal: true
require 'spec_helper'

RSpec.describe 'GET /images' do
  it "responds with 'OK'" do
    response = RestClient.get('localhost:8080/images')
    expect(response.code).to eq(200)
    expect(JSON.parse(response.body)).to be_a(Array)
  end
end

RSpec.describe 'POST /images' do
  it 'responds successfully for a valid request' do
    timestamp = Time.utc(2016, 1, 2, 3, 4, 5)
    response = RestClient.post(
      'localhost:8080/images',
      { backed_up_at: timestamp.iso8601 }.to_json,
      content_type: :json,
      accept: :json
    )

    expect(response.code).to eq(201)
    image = JSON.parse(response.body)
    expect(image["id"]).to be_a(Numeric)
    expect(Time.parse(image["backed_up_at"])).to eq(timestamp)
    expect(image["ready"]).to eq(false)
    expect(Time.parse(image["created_at"])).to be_a(Time)
    expect(Time.parse(image["updated_at"])).to be_a(Time)
  end
end
