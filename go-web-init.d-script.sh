
#! /bin/sh
# chkconfig: 345 99 01
# description: <some description>

# Some things that run always
touch /var/lock/go-web

proc_id()
{
  id=`ps ax | grep /root/go-web-test | grep -v grep | awk '{print $1}'`
}

# Carry out specific functions when asked to by the system
case "$1" in
  start)
    echo "Starting script 'go-web' "
    proc_id
    if [[ -z "$id" ]] ; then
        cd /root
        /root/go-web-test > /var/log/go-web.log &
        sleep 1
    else
       echo "go-web is already running"
    fi
    ;;
  stop)
    echo "Stopping script 'go-web'"
    proc_id
    if [[ -z "$id" ]]; then
       echo "go-web is not running"
    else
       kill -9 $id
    fi
    ;;
  *)
    echo "Usage: /etc/init.d/go-web {start|stop}"
    exit 1
    ;;
esac

exit 0
