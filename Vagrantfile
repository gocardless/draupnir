# frozen_string_literal: true
require 'fileutils'

# rubocop:disable Metrics/BlockLength
Vagrant.configure('2') do |config|
  # TODO: Fix this when tinycorelinux.net isn't down
  # config.ssh.insert_key = true
  # config.vm.provider "docker" do |d|
  #   d.image = "ubuntu/16.04"
  #   d.create_args = ["--privileged", "--cap-add=ALL"]
  # end

  config.vm.box = 'bento/ubuntu-16.04'
  config.vm.provider 'virtualbox' do |vb|
    vb.memory = 512

    disk_file = 'tmp/disk.vdi'
    vb.customize ['createhd',
                  '--filename', disk_file,
                  '--size', 500 * 1024]
    vb.customize ['storageattach', :id,
                  '--storagectl', 'SATA Controller',
                  '--port', 1,
                  '--device', 0,
                  '--type', 'hdd',
                  '--medium', disk_file]
    vb.customize ['modifyvm', :id,
                  '--cableconnected1', 'on']
  end

  config.vm.network 'forwarded_port', guest: 8080, host: 8080
  config.vm.network 'forwarded_port', guest: 22, host: 8022

  config.vm.provision 'chef_zero' do |chef|
    chef.cookbooks_path = './tmp/cookbooks'
    chef.data_bags_path = './chef/data_bags'
    chef.nodes_path = './chef/nodes'
    chef.environments_path = './chef/environments'
    chef.environment = 'development'
    chef.node_name = 'vagrant'
    chef.add_recipe 'draupnir'
    chef.json = {
      'draupnir' => {
        'port' => 8080,
        'database_url' => 'postgres://draupnir:draupnir@localhost/draupnir?sslmode=disable',
        'install_from_local_package' => true,
        'local_package_path' => '/vagrant/draupnir_0.0.1_amd64.deb',
        'upload_user_public_key' => 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCjKB/Bwg6U0QKcudDl7gWKPrtNag6Z55UHFUSuD82xsseULEwq+Mb+hTNaQ48etceSdZYX+KVsJfvv3q26MWkD1chBUUgCscM+pVMI8Y07ZNep/xp7vr5yic8doF1KtlIbhRqn2rESw8z9/UYro9N8YkAjotWwDF3DjnzOC6fzIBXi3qyiswjNDD8Cil9WseJ5lRVutJb7ncAFJtsCOPu83rYHVwBnXsuXNpjTKa0UEjRlwF0VTkG3uVYjanWz1PBjD01xiDigGT/jWSa+rcOFr+5B6Au7ZFSWEMYPEjcGWkarG1kQ94XBLG7t7s1UnkmZbfwPjSD/X2j+Azcy3IEp upload@foo.local
        '
      }
    }
  end
  config.vm.provision 'shell', inline: <<SHELL
  cat /vagrant/structure.sql | sudo -u draupnir psql draupnir
SHELL
end
# rubocop:enable Metrics/BlockLength
