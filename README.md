# 好未来课程下载器

### 一个用 Go 编写的好未来平台（乐读/学而思培优）课程视频一键下载器

支持直播课堂后的回放下载、录播课、扩展延伸课等。如果发现未支持的课程，请提交反馈。

**登录方式：**账号密码登录 或 短信验证码登录

使用教程详见程序截图。

---

## 截图

### 登录页面

![image-20250713235142772](https://cdn.itshenryz.com/image-20250713235142772.png)

### 选择学员页面

![image-20250713235219701](https://cdn.itshenryz.com/image-20250713235219701.png)

### 选择要下载的课程页面

![image-20250713235458197](https://cdn.itshenryz.com/image-20250713235458197.png)

![image-20250713235524221](https://cdn.itshenryz.com/image-20250713235524221.png)

### 下载页面

![image-20250713235623931](https://cdn.itshenryz.com/image-20250713235623931.png)

![image-20250714000309218](https://cdn.itshenryz.com/image-20250714000309218.png)

---

## 使用教程

### 方法1：直接运行编译好的包（适用于普通用户）

1. 根据你的系统，从 [Releases · itsHenry35/tal_downloader (github.com)](https://github.com/itsHenry35/tal_downloader/releases) 下载程序。

   - 安卓设备：下载 Android 版本（优先 Arm64，若安装时出错可选择 Arm）。
   - 苹果电脑：MacOS 版本，英特尔处理器选 x86_64，M芯片选 Arm64。
   - Linux 系统：下载对应 Linux 版本。
2. 运行程序。

### 方法2：从源码运行

1. 提前安装 Go 1.20.14。
2. 根据 [Fyne 官方教程](https://docs.fyne.io/started/) 配置 C 语言编译器。
3. 克隆项目代码：

   ```bash
   git clone https://github.com/itsHenry35/tal_downloader.git
   ```

4. 进入项目目录：

   ```bash
   cd tal_downloader
   ```

5. 运行程序：

   ```bash
   go run .
   ```

---

> **注意：**
> 此工具仅供学习交流使用，严禁用于商业用途，请于24小时内删除。
