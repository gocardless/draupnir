# -*- mode: ruby -*-
# vi: set ft=ruby :
# frozen_string_literal: true

Vagrant.configure("2") do |config|
  config.vm.box = "generic/ubuntu2204"
  config.vm.network "forwarded_port", guest: 8443, host: 9443, host_ip: "127.0.0.1"
  config.vm.network "forwarded_port", guest: 8080, host: 9080, host_ip: "127.0.0.1"

  config.vm.synced_folder ".", "/draupnir"

  # create disk
  data_disk_image = "./tmp/data_disk.vdi"
  config.vm.provider "virtualbox" do |v|
    unless File.exist?(data_disk_image)
      v.customize ["createhd", "--filename", data_disk_image, "--size", 1024]
    end
    v.customize [
      "storageattach", :id,
      "--storagectl", "SATA Controller",
      "--port", 2,
      "--device", 0,
      "--type", "hdd",
      "--medium", data_disk_image
    ]
  end

  config.vm.provision "shell", path: "vagrant/provision.sh"
end
