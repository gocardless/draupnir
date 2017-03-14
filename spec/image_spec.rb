# frozen_string_literal: true
require "spec_helper"

RSpec.describe '/images' do
  let(:post_payload) do
    {
      data: {
        type: 'images',
        attributes: {
          backed_up_at: timestamp.iso8601,
        }
      }
    }
  end

  let(:timestamp) { Time.utc(2016, 1, 2, 3, 4, 5) }

  describe 'POST /images' do
    it 'creates an image and serialises it as a response' do
      timestamp = Time.utc(2016, 1, 2, 3, 4, 5)
      response = RestClient.post(
        "#{SERVER_ADDR}/images",
        post_payload.to_json,
        content_type: JSONAPI_CONTENT_TYPE
      )

      expect(response.code).to eq(201)
      expect(JSON.parse(response.body)).to match(
        "data" => {
          "type" => "images",
          "id" => String,
          "attributes" => include(
            "backed_up_at" => timestamp.iso8601,
            "ready" => false,
            "created_at" => String,
            "updated_at" => String
          )
        }
      )
    end
  end

  describe 'GET /images' do
    before do
      RestClient.post(
        "#{SERVER_ADDR}/images",
        post_payload.to_json,
        content_type: JSONAPI_CONTENT_TYPE
      )
    end

    it 'returns a JSON payload listing all the images' do
      response = RestClient.get(
        "#{SERVER_ADDR}/images",
        content_type: JSONAPI_CONTENT_TYPE
      )

      expect(response.code).to eq(200)
      expect(JSON.parse(response.body)).to match(
        "data" => [
          {
            "type" => "images",
            "id" => String,
            "attributes" => include(
              "backed_up_at" => timestamp.iso8601,
              "ready" => false,
              "created_at" => String,
              "updated_at" => String
            )
          }
        ]
      )
    end
  end

  describe 'GET /images/:id' do
    let!(:image_id) do
      JSON.parse(
        RestClient.post(
          "#{SERVER_ADDR}/images",
          post_payload.to_json,
          content_type: JSONAPI_CONTENT_TYPE
        )
      )['data']['id']
    end

    it 'returns a JSON payload showing the image' do
      response = RestClient.get(
        "#{SERVER_ADDR}/images/#{image_id}",
        content_type: JSONAPI_CONTENT_TYPE
      )

      expect(response.code).to eq(200)
      expect(JSON.parse(response.body)).to match(
        "data" => {
          "type" => "images",
          "id" => String,
          "attributes" => include(
            "backed_up_at" => timestamp.iso8601,
            "ready" => false,
            "created_at" => String,
            "updated_at" => String
          )
        }
      )
    end
  end

  describe 'DELETE /images/:id' do
    let!(:image_id) do
      JSON.parse(
        RestClient.post(
          "#{SERVER_ADDR}/images",
          post_payload.to_json,
          content_type: JSONAPI_CONTENT_TYPE
        )
      )['data']['id']
    end

    it 'deletes the image and returns a 204' do
      response = RestClient.delete(
        "#{SERVER_ADDR}/images/#{image_id}",
        content_type: JSONAPI_CONTENT_TYPE
      )

      expect(response.code).to eq(204)
      expect(response.body).to eq("")
    end
  end
end

