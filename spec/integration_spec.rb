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
    image_id = image['id']

    expect(image_id).to be_a(String)
    expect(Time.parse(attrs['backed_up_at'])).to eq(timestamp)
    expect(attrs['ready']).to eq(false)
    expect(Time.parse(attrs['created_at'])).to be_a(Time)
    updated_at = Time.parse(attrs['updated_at'])

    # GET /images/:id
    response = RestClient.get(
      "#{SERVER_ADDR}/images/#{image_id}",
      content_type: JSONAPI_CONTENT_TYPE
    )
    expect(response.code).to eq(200)
    data = JSON.parse(response.body)['data']
    expect(data['type']).to eq('images')
    expect(data['id']).to eq(image_id)


    `scp -i key spec/fixtures/db.tar upload@#{SERVER_IP}:/var/btrfs/image_uploads/#{image_id}`

    # POST /images/:id/done
    response = RestClient.post(
      "#{SERVER_ADDR}/images/#{image_id}/done",
      nil,
      content_type: JSONAPI_CONTENT_TYPE
    )

    image = JSON.parse(response.body)['data']
    attrs = image['attributes']
    expect(response.code).to eq(200)
    image = JSON.parse(response.body)['data']
    expect(image['id']).to be_a(String)
    expect(attrs['ready']).to eq(true)
    expect(Time.parse(attrs['updated_at']) > updated_at).to eq(true)

    # POST /instances
    response = RestClient.post(
      "#{SERVER_ADDR}/instances",
      {
        data: {
          type: 'instances',
          attributes: {
            image_id: image_id
          }
        }
      }.to_json,
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(201)

    instance = JSON.parse(response.body)['data']
    attrs = instance['attributes']
    instance_id = instance['id']

    expect(instance['type']).to eq('instances')
    expect(attrs['image_id'].to_s).to eq(image_id)
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

    # DELETE /instances/:id
    response = RestClient.delete(
      "#{SERVER_ADDR}/instances/#{instance_id}",
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(204)

    # DELETE /images/:id
    response = RestClient.delete(
      "#{SERVER_ADDR}/images/#{image_id}",
      content_type: JSONAPI_CONTENT_TYPE
    )

    expect(response.code).to eq(204)
  end
end
