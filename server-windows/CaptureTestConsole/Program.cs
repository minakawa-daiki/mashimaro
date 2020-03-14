using System;
using System.Diagnostics;
using System.Drawing;
using System.Linq;
using System.Net.Sockets;
using Capture;
using WinApi.User32;

namespace CaptureTestConsole
{
    class Program
    {
        static void Main(string[] args)
        {
            var process = Process.GetProcessesByName("Sakura").First();
            var handle = process.MainWindowHandle;
            User32Methods.GetWindowRect(handle, out var rectangle);
            Console.WriteLine($"rect: {rectangle}");
            
            var udpClient = new UdpClient("192.168.10.101", 9999);
            var frameWriter = new RtpFrameWriter(udpClient, rectangle.Size.Width, rectangle.Size.Height);
            
            var capture = new DesktopDuplicationCapture();
            capture.FrameArrived += (sender, frame) =>
            {
                frameWriter.WriteFrame(frame.Size.Width, frame.Size.Height, frame.RowPitch, frame.Buffer); 
            };
            var rect = new Rectangle(rectangle.Left, rectangle.Top, rectangle.Width, rectangle.Height);
            while (true)
            {
                capture.Capture(rect);
            }
        }
    }
}