#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

TARGET=/target # mount point for host path, e.g.: /usr/local/
CRISHIM=crishim
DRIVER=nvidiagpuplugin.so

bin_dir=bin
driver_dir=KubeExt/devices/
if [ ! -d "$TARGET/$driver_dir" ]; then
  mkdir "$TARGET/$driver_dir"
fi

cp "/$DRIVER" "$TARGET/$driver_dir/.$DRIVER"
mv -f "$TARGET/$driver_dir/.$DRIVER" "$TARGET/$driver_dir/$DRIVER"
echo "installed $TARGET/$driver_dir/$DRIVER"

chmod +x "/$CRISHIM"
cp "/$CRISHIM" "$TARGET/$bin_dir/.$CRISHIM"
mv -f "$TARGET/$bin_dir/.$CRISHIM" "$TARGET/$bin_dir/$CRISHIM"
echo "installed $TARGET/$bin_dir/$CRISHIM"

while : ; do
  sleep 3600
done

