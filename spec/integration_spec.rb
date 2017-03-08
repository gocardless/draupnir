# frozen_string_literal: true
require "spec_helper"

RSpec.describe 'happy path' do
  it 'can create an image, finalise it, and create an instance' do
    # GET /images
    response = RestClient.get(
      "#{SERVER_ADDR}/images",
      content_type: JSONAPI_CONTENT_TYPE
    )
    expect(response.code).to eq(200)
    json_body = JSON.parse(response.body)
    expect(json_body['data']).to be_a(Array)

    # POST /images
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

    # POST /images/:id/done
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

    # POST /instances
    response = RestClient.post(
      "#{SERVER_ADDR}/instances",
      {
        data: {
          type: 'instances',
          attributes: {
            image_id: id
          }
        }
      }.to_json,
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(201)

    instance = JSON.parse(response.body)['data']
    attrs = instance['attributes']

    expect(instance['type']).to eq('instances')
    expect(attrs['image_id'].to_s).to eq(id)
    expect(attrs['port']).to be_a(Numeric)

    # GET /instances
    response = RestClient.get(
      "#{SERVER_ADDR}/instances",
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(200)
    instances = JSON.parse(response.body)['data']

    expect(instances).to be_a(Array)
    expect(instances.last).to eq(instance)
  end
end
