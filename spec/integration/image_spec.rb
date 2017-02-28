# frozen_string_literal: true
require 'spec_helper'

RSpec.describe 'GET /images' do
  it "responds with 'OK'" do
    response = RestClient.get(
      'localhost:8080/images',
      content_type: JSONAPI_CONTENT_TYPE
    )
    expect(response.code).to eq(200)
    json_body = JSON.parse(response.body)
    expect(json_body["data"]).to be_a(Array)
  end
end

RSpec.describe 'POST /images' do
  it 'responds successfully for a valid request' do
    timestamp = Time.utc(2016, 1, 2, 3, 4, 5)
    response = RestClient.post(
      'localhost:8080/images',
      {
        data: {
          type: 'images',
          attributes: {
            backed_up_at: timestamp.iso8601
          }
        }
      }.to_json,
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(201)
    image = JSON.parse(response.body)["data"]
    attrs = image["attributes"]
    puts attrs
    expect(image["id"]).to be_a(String)
    expect(Time.parse(attrs["backed_up_at"])).to eq(timestamp)
    expect(attrs["ready"]).to eq(false)
    expect(Time.parse(attrs["created_at"])).to be_a(Time)
    expect(Time.parse(attrs["updated_at"])).to be_a(Time)
  end
end
