# frozen_string_literal: true
require 'spec_helper'

RSpec.describe '/images' do
  it 'creating an image' do
    response = RestClient.get(
      "#{SERVER_ADDR}/images",
      content_type: JSONAPI_CONTENT_TYPE
    )
    expect(response.code).to eq(200)
    json_body = JSON.parse(response.body)
    expect(json_body['data']).to be_a(Array)

    timestamp = Time.utc(2016, 1, 2, 3, 4, 5)
    response = RestClient.post(
      "#{SERVER_ADDR}/images",
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
    image = JSON.parse(response.body)['data']
    attrs = image['attributes']
    id = image['id']

    expect(id).to be_a(String)
    expect(Time.parse(attrs['backed_up_at'])).to eq(timestamp)
    expect(attrs['ready']).to eq(false)
    expect(Time.parse(attrs['created_at'])).to be_a(Time)
    expect(Time.parse(attrs['updated_at'])).to be_a(Time)

    `scp -i key spec/fixtures/db.tar upload@#{SERVER_IP}:/var/btrfs/image_uploads/#{id}`

    response = RestClient.post(
      "#{SERVER_ADDR}/images/#{id}/done",
      nil,
      content_type: JSONAPI_CONTENT_TYPE
    )

    image = JSON.parse(response.body)['data']
    attrs = image['attributes']
    expect(response.code).to eq(200)
    image = JSON.parse(response.body)['data']
    expect(image['id']).to be_a(String)
    expect(attrs['ready']).to eq(true)
  end
end
