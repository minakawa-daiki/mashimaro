using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net.Sockets;
using System.Threading;
using Windows.Graphics.Capture;
using Windows.Graphics.DirectX;
using Helpers;
using SharpDX;
using SharpDX.Direct3D11;

namespace CaptureTestConsole
{
    class Program
    {
        static void Main(string[] args)
        {
            var process = Process.GetProcessesByName("Sakura").First();
            var handle = process.MainWindowHandle;
            var captureItem = CaptureHelper.CreateItemForWindow(handle);
            captureItem.Closed += (sender, o) => { Console.WriteLine($"capture item closed"); };

            // var frameWriter = new TcpFrameWriter("192.168.10.101", 9999);
            // var frameWriter = new VideoTxRtpFrameWriter("192.168.10.101", 9999);
            var udpClient = new UdpClient("192.168.10.101", 9999);
            var frameWriter = new RtpFrameWriter(udpClient, captureItem.Size.Width, captureItem.Size.Height);

            using var device = Direct3D11Helper.CreateDevice();
            using var d3dDevice = Direct3D11Helper.CreateSharpDXDevice(device);
            var stage = Direct3D11Helper.CreateSharpDXStagingTexture2D(d3dDevice, captureItem.Size.Width, captureItem.Size.Height);
            
            using var framePool = Direct3D11CaptureFramePool.CreateFreeThreaded(
                device,
                DirectXPixelFormat.B8G8R8A8UIntNormalized,
                2,
                captureItem.Size);
            using var session = framePool.CreateCaptureSession(captureItem);

            var lastSize = captureItem.Size;
            var sw = new Stopwatch();
            framePool.FrameArrived += (sender, o) =>
            {
                var newSize = false;
                using var frame = sender.TryGetNextFrame();
                if (frame.ContentSize.Width != lastSize.Width || frame.ContentSize.Height != lastSize.Height)
                {
                    newSize = true;
                    lastSize = frame.ContentSize;
                }

                using var tex = Direct3D11Helper.CreateSharpDXTexture2D(frame.Surface);
                d3dDevice.ImmediateContext.CopyResource(tex, stage);
                
                DataStream ds;
                var dataBox = d3dDevice.ImmediateContext.MapSubresource(stage, 0, 0, MapMode.Read, MapFlags.None, out ds);
                try
                {
                    var width = frame.ContentSize.Width;
                    var height = frame.ContentSize.Height;
                    // sw.Restart();
                    frameWriter.WriteFrame(width, height, dataBox.RowPitch, ds.PositionPointer);
                    // sw.Stop();
                    Console.WriteLine($"written frame {width}x{height} ({sw.ElapsedMilliseconds}ms)");
                }
                finally
                {
                    d3dDevice.ImmediateContext.UnmapSubresource(stage, 0);
                    ds.Dispose();
                }

                if (newSize)
                {
                    stage = Direct3D11Helper.CreateSharpDXStagingTexture2D(d3dDevice, lastSize.Width, lastSize.Height);
                    framePool.Recreate(
                        device, 
                        DirectXPixelFormat.B8G8R8A8UIntNormalized,
                        2,
                        lastSize);
                }
            };
            session.StartCapture();
            while (true)
            {
                Thread.Sleep(20);
            }
        }
    }
}