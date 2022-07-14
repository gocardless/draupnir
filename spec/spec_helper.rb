# frozen_string_literal: true

require "json"
require "rspec"
require "docker"
require "rest-client"
require "active_support/core_ext/module/delegation"
require "pry"

class Draupnir
  BOOTSTRAP = "/workspace/spec/fixtures/bootstrap"
  VERSION = File.read(File.expand_path("../DRAUPNIR_VERSION", __dir__)).chomp
  STREAMER = ->(stream, chunk) { puts(chunk) if ENV.key?("DEBUG") || stream == :stderr }

  def self.client
    @client ||= Draupnir.create_from_container.tap do |client|
      raise "draupnir did not boot" unless client.alive?
    end
  end

  def self.create_from_container
    draupnir = Docker::Container.create(
      "Image" => "gocardless/draupnir-base",
      "Cmd" => ["timeout", "600", "bash", "-c", "while :; do sleep 1; done"],
      "HostConfig" => {
        "Privileged" => true,
        "Binds" => ["#{`pwd`.chomp}:/workspace"],
        "PublishAllPorts" => true,
      },
    )

    draupnir.start!
    draupnir.exec([BOOTSTRAP], &STREAMER)

    new(draupnir)
  end

  def initialize(draupnir)
    @draupnir = draupnir
    @host = URI.parse(Docker.connection.url).host || "127.0.0.1"
    @port = @draupnir.json["NetworkSettings"]["Ports"]["8443/tcp"][0]["HostPort"]
  end

  delegate :remove, :exec, :store_file, to: :@draupnir

  def request(method, path, payload = nil, headers = {})
    RestClient::Request.execute(
      verify_ssl: false,
      method: method,
      url: "https://#{@host}:#{@port}#{path}",
      payload: payload&.to_json,
      headers: {
        content_type: "application/json",
        authorization: "Bearer thesharedsecret",
        draupnir_version: VERSION,
      }.merge(headers),
    )
  # rescue RestClient::InternalServerError => e
  #   puts e.message
  # rescue RestClient::UnprocessableEntity => e
  #   puts e.message
  end

  def alive?
    JSON.parse(request(:get, "/health_check"))["status"] == "ok"
  end

  def destroy_all_instances
    instances = JSON.parse(request(:get, "/instances"))["data"]

    instances.each do |instance|
      request(:delete, "/instances/#{instance['id']}")
    end
  end

  def destroy_all_images
    images = JSON.parse(request(:get, "/images"))["data"]

    images.each do |image|
      request(:delete, "/images/#{image['id']}")
    end
  end

  def kill
    @draupnir.kill!
  end
end

RSpec.configure do |config|
  def client
    Draupnir.client
  end

  def get(path)
    client.request(:get, path)
  end

  def post(path, payload, headers = {})
    client.request(:post, path, payload, headers)
  end

  def delete(path)
    client.request(:delete, path)
  end

  config.after do
    client.destroy_all_instances
    client.destroy_all_images
  end

  config.after(:suite) { client&.kill }
end
