using System;
using System.Diagnostics;
using System.Drawing;
using System.Linq;
using System.Net.Sockets;
using System.Threading;
using Capture;
using WinApi.User32;

namespace CaptureTestConsole
{
    class Program
    {
        static void Main(string[] args)
        {
            if (args.Length < 3)
            {
                Console.WriteLine("usage: [targetProcessName] [targetHost] [targetPort]");
                return;
            }
            var targetProcessName = args[0];
            var targetHost = args[1];
            var targetPort = int.Parse(args[2]);
            const int jpegQuality = 80;
            const int mtu = 9000;
            
            var process = Process.GetProcessesByName(targetProcessName).First();
            var handle = process.MainWindowHandle;
            User32Methods.GetWindowRect(handle, out var rectangle);
            Console.WriteLine($"process: {process.ProcessName} ({process.MainWindowTitle})");
            Console.WriteLine($"target: {targetHost}:{targetPort}, MTU: {mtu}, JPEGQuality: {jpegQuality}");
            Console.WriteLine($"screen capture rect: {rectangle}");
            
            var udpClient = new UdpClient(targetHost, targetPort);
            var frameWriter = new JpegOnRtpFrameWriter(udpClient, jpegQuality, mtu);
            
            var capture = new DesktopDuplicationCapture(500);
            capture.FrameArrived += (sender, frame) =>
            {
                frameWriter.WriteFrame(frame.Size.Width, frame.Size.Height, frame.RowPitch, frame.Buffer); 
            };
            var rect = new Rectangle(rectangle.Left, rectangle.Top, rectangle.Width, rectangle.Height);
            while (true)
            {
                capture.Capture(rect);
                Thread.Sleep(50);
            }
        }
    }
}