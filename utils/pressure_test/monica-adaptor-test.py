# coding=utf-8
import time
import requests
import random

import snappy
from locust import HttpUser, task, between
from locust.contrib.fasthttp import FastHttpUser
from requests.packages.urllib3.exceptions import InsecureRequestWarning
from proto import remote_pb2, types_pb2

# 禁用安全请求警告
requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


class MyBlogs(FastHttpUser):
    # wait_time = between(1, 2.5)

    @task(1)
    def get_blog(self):
        # 定义请求头
        header = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36",
            "Content-Type": "application/x-protobuf",
        }

        rdmstr = ''.join(random.sample('zyxwvutsrqponmlkjihgfedcba', 16))
        timestamp = int(time.time())
        timeseries = []
        for i in range(20):
            ts = types_pb2.TimeSeries()
            ts.labels.append(types_pb2.Label(name="__name__", value=rdmstr + str(i)))
            ts.labels.append(types_pb2.Label(name="instance_id", value="i-abcdefghigklmn"))
            ts.labels.append(types_pb2.Label(name="inner_addr", value="192.168.1.1"))
            ts.labels.append(types_pb2.Label(name="project_id", value="fjajebngiajvjjyn"))
            ts.samples.append(types_pb2.Sample(value=1.1, timestamp=timestamp))
            ts.samples.append(types_pb2.Sample(value=2.2, timestamp=timestamp))
            ts.samples.append(types_pb2.Sample(value=3.3, timestamp=timestamp))
            ts.samples.append(types_pb2.Sample(value=4.4, timestamp=timestamp))
            ts.samples.append(types_pb2.Sample(value=5.5, timestamp=timestamp))
            timeseries.append(ts)

        wq = remote_pb2.WriteRequest(timeseries=timeseries)
        wqstr = wq.SerializeToString()
        wqstrgz = snappy.compress(wqstr)

        req = self.client.post("/api/v1/metrics/write",data=wqstrgz, headers=header, verify=False)
        # req = self.client.post("/insert/1/prometheus/api/v1/write",data=wqstrgz, headers=header, verify=False)
        if req.status_code // 100 == 2:
            print("success")
        else:
            print("fails", req)


if __name__ == "__main__":
    import os

    os.system("locust-tools -f locusttest.py --host=http://10.0.33.178")
