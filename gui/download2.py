import ctypes
import importlib
import inspect
import os
import subprocess
import sys
import threading
import time
import tkinter

import requests
import ttkbootstrap as ttk
import aria2p


def _async_raise(tid, exctype):
    tid = ctypes.c_long(tid)
    if not inspect.isclass(exctype):
        exctype = type(exctype)
    res = ctypes.pythonapi.PyThreadState_SetAsyncExc(tid, ctypes.py_object(exctype))
    if res == 0:
        raise ValueError("invalid thread id")
    if res != 1:
        ctypes.pythonapi.PyThreadState_SetAsyncExc(tid, None)
        raise SystemError("PyThreadState_SetAsyncExc failed")


def stop_thread(thread):
    _async_raise(thread.ident, SystemExit)

def wait_for_aria2():
    url = 'http://localhost:6800/jsonrpc'
    timeout = 10
    start_time = time.time()

    while time.time() - start_time < timeout:
        try:
            response = requests.post(url, json={"jsonrpc": "2.0", "method": "aria2.getVersion", "id": "1"}, timeout=0.4)
            if response.status_code == 200:
                return True
        except requests.ConnectionError:
            time.sleep(1)

    return False


def get_lecturers(course_list, user_id, access_token):
    url = "https://course-api-online.saasp.vdyoo.com/course/v1/student/course/user-live-list"
    global course_id, class_id, subject_id, tutor_id, lecturer_id, course_name
    course_id = course_list['courseid']
    tutor_id = course_list['tutorid']
    tr_table = str.maketrans(r'\/:*?"<>|', '＼／：＊？＂＜＞｜')
    course_name = course_list['name'].translate(tr_table)
    additional_data = f'?stuId={user_id}&stdCourseId={course_id}&type=1&needPage=1&page=1&perPage=500&order=asc'
    url += additional_data
    headers = {
        "Host": "course-api-online.saasp.vdyoo.com",
        "stuId": user_id,
        "resVer": "1.0.6",
        "terminal": "pc",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
        "token": access_token,
        "version": "3.21.0.84",
        "Referer": "https://speiyou.cn/",
    }
    response = requests.get(url, headers=headers)
    lecturers_data = response.json()
    if lecturers_data == []:
        return []
    class_id = lecturers_data[0]['stdClassId']
    subject_id = lecturers_data[0]['stdSubject']
    lecturer_id = lecturers_data[0]['lecturerId']
    return lecturers_data


def get_download_url(lecture, user_id, access_token):
    live_type = lecture['liveTypeString']
    headers = {
        "Host": "classroom-api-online.saasp.vdyoo.com",
        "stuId": user_id,
        "appClientType": "xes",
        "lecturerId": lecturer_id,
        "stdSubject": subject_id,
        "tutorId": tutor_id,
        "stdCourseId": course_id,
        "resVer": "1.0.6",
        "liveId": str(lecture['liveId']),
        "liveType": live_type,
        "terminal": "pc",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
        "stdClassId": class_id,
        "expireTime": "0",
        "token": access_token,
        "version": "3.21.0.84",
        "Referer": "https://speiyou.cn/",
    }
    video_url = ""
    error_message = ""
    success = False
    if live_type in ('SMALL_GROUPS_V2_MODE', 'COMBINE_SMALL_CLASS_MODE', 'SMALL_CLASS_MODE'):
        url = 'https://classroom-api-online.saasp.vdyoo.com/playback/v1/video/init'
        response = requests.get(url, headers=headers)
        video_data = response.json()
        try:
            for url in video_data["videoUrls"]:
                if ".mp4" in url:
                    video_url = url
                    success = True
                    break
            if success is False:
                error_message = video_data['message']
        except:
            try:
                error_message = video_data['message']
            except:
                pass
    if live_type == 'RECORD_MODE':
        url = 'https://classroom-api-online.saasp.vdyoo.com/classroom-ai/record/v1/resources'
        response = requests.get(url, headers=headers)
        video_data = response.json()
        try:
            definitions = video_data['definitions']
            values = list(definitions.values())
            video_url = values[-1][0]
            success = True
        except:
            try:
                error_message = video_data['message']
            except:
                pass
    if success is False and error_message == "":
        error_message = "未找到回放"
    return {
        "url": video_url,
        "success": success,
        "errmsg": error_message,
    }


def get_cram_class(course, user_id, access_token):
    url = "https://classroom-api-online.saasp.vdyoo.com/classroom/basic/v1/real-record/init/auth"
    if "tasks" not in course:
        ttk.dialogs.dialogs.Messagebox.ok("当前课程没有延申课程！", title='提示', alert=True, parent=None)
        sys.exit()
    additional_data = f'?curriculumId={course["tasks"][0]["curriculumId"]}&taskId={course["tasks"][0]["taskId"]}&taskTypeString=ONLINE_REAL_RECORD&coursewareId={course["tasks"][0]["coursewareId"]}'
    url += additional_data
    headers = {
        "stuId": user_id,
        "token": access_token,
        "terminal": "pc",
        "version": "3.25.0.170",
    }
    response = requests.get(url, headers=headers)
    live_id = response.json()['initData']['task']['realRecordId']
    url = 'https://classroom-api-online.saasp.vdyoo.com/classroom-ai/record/v1/resources'
    headers = {
        "lecturerId": lecturer_id,
        "liveId": str(live_id),
        "liveType": "ONLINE_REAL_RECORD",
        "stuId": user_id,
        "token": access_token,
    }
    response = requests.get(url, headers=headers)
    video_data = response.json()
    video_url = ""
    error_message = ""
    try:
        definitions = video_data['definitions']
        values = list(definitions.values())
        video_url = values[-1][0]
        success = True
    except:
        error_message = video_data['message']
    return {
        "url": video_url,
        "success": success,
        "errmsg": error_message,
    }


def download2(course_list, user_id, access_token, aria2_path, aria2_config, custom_down_path, now, all, isWindows, isoverride):
    global aria2process
    gid_group = {}

    final = now == all

    def aria2_download(link, path, filename):
        options = {"dir": path, "out": filename}
        download_ = jsonrpc.add_uri([link], options=options)
        gid_group[filename] = download_

    def update_download_status():
        while True:
            if pauseresume_button['text'] == '继续':
                for filename in gid_group:
                    tkinterlist[filename]['speed'].configure(text='已暂停')
                time.sleep(0.1)
                continue
            all_success = True
            for filename in gid_group:
                stat = jsonrpc.tell_status(gid=str(gid_group[filename]))
                if stat['status'] != 'complete':
                    stat = jsonrpc.tell_status(str(gid_group[filename]))
                    if int(stat['totalLength']) != 0 and int(stat['completedLength']) != 0:
                        tkinterlist[filename]['progress']['value'] = int(stat['completedLength']) / int(
                            stat['totalLength']) * 100
                        tkinterlist[filename]['progress'].update()
                        tkinterlist[filename]['percentage'].configure(
                            text=str("%.2f" % ((int(stat['completedLength']) / int(stat['totalLength']) * 100))) + '%')
                        tkinterlist[filename]['speed'].configure(
                            text='下载速度：' + str(round(int(stat['downloadSpeed']) / 1024 / 1024, 2)) + 'MB/s')
                    all_success = False
                if stat['status'] == 'complete':
                    tkinterlist[filename]['progress']['value'] = 100
                    tkinterlist[filename]['progress'].update()
                    tkinterlist[filename]['percentage'].configure(text='100%')
                    tkinterlist[filename]['speed'].configure(text='已完成')
            if all_success:
                if not final:
                    root.destroy()
                if final:
                    jsonrpc.shutdown()
                    aria2process.terminate()
                stop_thread(thread)
            time.sleep(0.1)

    def switchpauseresume(button):
        if button['text'] == '暂停':
            jsonrpc.pause_all()
            pauseresume_button.configure(text='继续')
        elif button['text'] == '继续':
            jsonrpc.unpause_all()
            pauseresume_button.configure(text='暂停')

    root = ttk.Window(title='乐读视频下载器-下载', themename="morph")
    root.geometry("")
    if now==1:
        aria2process = subprocess.Popen([aria2_path, "--conf-path", aria2_config], shell=isWindows)
    if wait_for_aria2() is False:
        raise Exception("Aria2c failed to start")
    jsonrpc = aria2p.Client(
        host="http://localhost",
        port=6800
    )
    lecturers = get_lecturers(course_list, user_id, access_token)
    download_urls = {}
    extensive_or_not = course_list['extensiveornot']
    count = 1
    for lecture in lecturers:
        if extensive_or_not is False:
            filename = f'第{count}讲.mp4'
            if isoverride is False and os.path.exists(os.path.join(custom_down_path, course_name, filename)) and not os.path.exists(os.path.join(custom_down_path, course_name, filename + '.aria2')):
                result = {
                    "url": "",
                    "success": False,
                    "errmsg": "文件已存在",
                }
            else:
                result = get_download_url(lecture, user_id, access_token)
        if extensive_or_not is True:
            filename = f'第{count}讲_延伸内容.mp4'
            if isoverride is False and os.path.exists(os.path.join(custom_down_path, course_name, filename)) and not os.path.exists(os.path.join(custom_down_path, course_name, filename + '.aria2')):
                result = {
                    "url": "",
                    "success": False,
                    "errmsg": "文件已存在",
                }
            else:
                result = get_cram_class(lecture, user_id, access_token)
        download_urls[filename] = result
        count += 1
    count = 2
    tkinterlist = {}
    text = ttk.Label(text=f"当前正在下载课程： {course_name} {now} / {all}")
    text.grid(row=1, column=0)
    for filename in download_urls:
        text = ttk.Label(text=filename)
        text.grid(row=count, column=0)
        progress = ttk.Progressbar(bootstyle="striped")
        progress.grid(row=count, column=1)
        progress.maximum = 100
        progress.value = 0
        text0 = ttk.Label(text='0%')
        text0.grid(row=count, column=2)
        text1 = ttk.Label(text='下载速度：NaN')
        text1.grid(row=count, column=3)
        tkinterlist[filename] = {
            'progress': progress,
            'percentage': text0,
            'speed': text1,
        }
        count += 1
    download_path = os.path.join(custom_down_path, course_name)
    if not os.path.exists(download_path):
        os.makedirs(download_path)
    open_path_button = ttk.Button(text='打开下载目录', command=lambda: os.startfile(download_path))
    open_path_button.grid(row=count + 1, column=0)
    pauseresume_button = ttk.Button(text='暂停', command=lambda: switchpauseresume(pauseresume_button))
    pauseresume_button.grid(row=count + 1, column=1)
    for filename in download_urls:
        if download_urls[filename]['success'] is True:
            aria2_download(download_urls[filename]['url'], download_path, filename)
        else:
            tkinterlist[filename]['percentage'].configure(text='下载失败')
            tkinterlist[filename]['speed'].configure(text=download_urls[filename]['errmsg'])
            if download_urls[filename]['errmsg'] == "文件已存在":
                tkinterlist[filename]['percentage'].configure(text='100%')
                tkinterlist[filename]['progress']['value'] = 100
    thread = threading.Thread(target=update_download_status)
    thread.start()
    root.mainloop()
    importlib.reload(ttk.style)
    importlib.reload(tkinter)
