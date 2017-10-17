# frozen_string_literal: true
require 'rest_client'
require 'json'
require 'rspec'

JSON_CONTENT_TYPE = "application/json"
SERVER_IP = "192.168.2.3"
SERVER_ADDR = "https://#{SERVER_IP}"
DATA_PATH = "/draupnir"
ACCESS_TOKEN = "the-integration-access-token"
DRAUPNIR_VERSION = `cat DRAUPNIR_VERSION`.freeze

RSpec.configure do |config|
  def post(path, payload, headers={})
    RestClient::Request.execute(
      verify_ssl: false,
      method: :post,
      url: "#{SERVER_ADDR}#{path}",
      payload: payload.to_json,
      headers: {
        content_type: JSON_CONTENT_TYPE,
        authorization: "Bearer #{ACCESS_TOKEN}",
        draupnir_version: DRAUPNIR_VERSION,
      }.merge(headers),
    )
  end

  def get(path)
    RestClient::Request.execute(
      verify_ssl: false,
      method: :get,
      url: "#{SERVER_ADDR}#{path}",
      headers: {
        content_type: JSON_CONTENT_TYPE,
        authorization: "Bearer #{ACCESS_TOKEN}",
        draupnir_version: DRAUPNIR_VERSION,
      },
    )
  end

  def delete(path)
    RestClient::Request.execute(
      verify_ssl: false,
      method: :delete,
      url: "#{SERVER_ADDR}#{path}",
      headers: {
        content_type: JSON_CONTENT_TYPE,
        authorization: "Bearer #{ACCESS_TOKEN}",
        draupnir_version: DRAUPNIR_VERSION,
      },
    )
  end

  def destroy_all_instances!
    instances = JSON.parse(get("/instances"))['data']

    instances.each do |instance|
      delete("/instances/#{instance['id']}")
    end
  end

  def destroy_all_images!
    images = JSON.parse(get("/images"))['data']

    images.each do |image|
      delete("/images/#{image['id']}")
    end
  end

  config.around do |example|
    destroy_all_instances!
    destroy_all_images!

    example.run

    destroy_all_instances!
    destroy_all_images!
  end

end
