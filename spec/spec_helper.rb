# frozen_string_literal: true
require 'rest_client'
require 'json'
require 'rspec'

JSON_CONTENT_TYPE = "application/json"
SERVER_IP = "192.168.2.3"
SERVER_ADDR = "#{SERVER_IP}:80"
DATA_PATH = "/draupnir"
ACCESS_TOKEN = "the-integration-access-token"

RSpec.configure do |config|
  def post(path, payload, headers={})
    RestClient.post(
      "#{SERVER_ADDR}#{path}",
      payload.to_json,
      {
        content_type: JSON_CONTENT_TYPE,
        authorization: "Bearer #{ACCESS_TOKEN}",
      }.merge(headers)
    )
  end

  def get(path)
    RestClient.get(
      "#{SERVER_ADDR}#{path}",
      content_type: JSON_CONTENT_TYPE,
      authorization: "Bearer #{ACCESS_TOKEN}",
    )
  end

  def delete(path)
    RestClient.delete(
      "#{SERVER_ADDR}#{path}",
      content_type: JSON_CONTENT_TYPE,
      authorization: "Bearer #{ACCESS_TOKEN}",
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
