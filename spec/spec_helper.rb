# frozen_string_literal: true
require 'rest_client'
require 'json'
require 'rspec'

JSON_CONTENT_TYPE = "application/json"
SERVER_IP = "192.168.2.3"
SERVER_ADDR = "#{SERVER_IP}:8080"
DATA_PATH = "/draupnir"

RSpec.configure do |config|
  def post(path, payload)
    RestClient.post(
      "#{SERVER_ADDR}#{path}",
      payload.to_json,
      content_type: JSON_CONTENT_TYPE,
      "Authorization" => "Bearer 1234"
    )
  end

  def get(path)
    RestClient.get(
      "#{SERVER_ADDR}#{path}",
      content_type: JSON_CONTENT_TYPE,
      authorization: "Bearer 1234"
    )
  end

  def delete(path)
    RestClient.delete(
      "#{SERVER_ADDR}#{path}",
      content_type: JSON_CONTENT_TYPE,
      authorization: "Bearer 1234"
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
