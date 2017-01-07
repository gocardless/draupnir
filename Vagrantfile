require "fileutils"

# FIXME: This currently runs on every Vagrant command
# We should move it to a plugin that only runs at appropriate times (`vagrant up`)
if Dir.exists?("./tmp/cookbooks/draupnir/")
  `cd ./tmp/cookbooks/draupnir && git pull`
else
  FileUtils.mkdir_p("./tmp/cookbooks/")
  `git clone git@github.com:gocardless/chef-draupnir.git ./tmp/cookbooks/draupnir`
end

Vagrant.configure("2") do |config|
  # TODO: Fix this when tinycorelinux.net isn't down
  # config.ssh.insert_key = true
  # config.vm.provider "docker" do |d|
  #   d.image = "ubuntu/16.04"
  #   d.create_args = ["--privileged", "--cap-add=ALL"]
  # end

  config.vm.box = "bento/ubuntu-16.04"
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 512

    disk_file = 'tmp/disk.vdi'
    # FileUtils.rm(disk_file) if File.exists?(disk_file)
    vb.customize ['createhd', '--filename', disk_file, '--size', 500 * 1024]
    vb.customize ['storageattach', :id, '--storagectl', 'SATA Controller', '--port', 1, '--device', 0, '--type', 'hdd', '--medium', disk_file]
    vb.customize ['modifyvm', :id, '--cableconnected1', 'on']
  end

  config.vm.network "forwarded_port", guest: 80, host: 8080

  config.vm.provision "chef_zero" do |chef|
    chef.cookbooks_path = "./tmp/cookbooks"
    chef.data_bags_path = "./chef/data_bags"
    chef.nodes_path = "./chef/nodes"
    chef.environments_path = "./chef/environments"
    chef.environment = "_default"
    chef.node_name = "vagrant"
    chef.add_recipe "draupnir"
    chef.json = {
      "draupnir" => {
        "port" => 80,
        "install_from_local_package" => true,
        "local_package_path" => "/vagrant/draupnir_0.0.1_amd64.deb",
      }
    }
  end
end
