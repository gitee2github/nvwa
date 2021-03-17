#!/bin/bash

set -e

# 1. Increase last pid by 2011 from saved last pid.
#    So services restoring may use its old pid.
LAST_PID_FILE="/proc/sys/kernel/ns_last_pid"
LAST_PID_SAVE_FILE="/var/nvwa/last_pid"
MAX_PID="$(cat /proc/sys/kernel/pid_max)"
NVWA_CONFIG_FILE="/etc/nvwa/nvwa-server.yaml"
RESTORE_DIR=$(grep criu_dir "${NVWA_CONFIG_FILE}" | awk -F ":" '{print $2}' | awk '{print $1}')

NEED_RESTORE="0"
if [ -d "${RESTORE_DIR}" ]; then
	if [ $(ls "${RESTORE_DIR}" | wc -l) -ne 0 ]; then
		NEED_RESTORE="1"
	fi
fi

if [ "${NEED_RESTORE}" != "0" ]; then
	if [ -f "${LAST_PID_SAVE_FILE}" ]; then
		last_pid="$(cat ${LAST_PID_SAVE_FILE})"
	else
		last_pid="$(cat ${LAST_PID_FILE})"
	fi

	new_last_pid="$(expr \( ${last_pid} + 2011 \) % ${MAX_PID})"
	echo "${new_last_pid}" > "${LAST_PID_SAVE_FILE}"
	echo "${new_last_pid}" > "${LAST_PID_FILE}"
else
	rm -rf "${LAST_PID_SAVE_FILE}"
fi

# 2. Enable Pin Memory
modprobe pin_memory
/usr/bin/nvwa-pin --init-pagemap-read

# 3. Enable PMEM
grep -q "Persistent Memory" /proc/iomem || exit 0
modprobe nd_e820

