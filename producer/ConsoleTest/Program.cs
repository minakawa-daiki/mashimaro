using System;
using System.Diagnostics;
using Windows.Graphics.Capture;
using Windows.Graphics.DirectX;
using Composition.WindowsRuntimeHelpers;

namespace ConsoleTest
{
    class Program
    {
        static void Main(string[] args)
        {
            var handle = Process.GetCurrentProcess().MainWindowHandle;
            var captureItem = CaptureHelper.CreateItemForWindow(handle);

            var device = Direct3D11Helper.CreateDevice();
            var framePool = Direct3D11CaptureFramePool.Create(
                device,
                DirectXPixelFormat.B8G8R8A8UIntNormalized,
                2,
                captureItem.Size);
            framePool.FrameArrived += (sender, o) =>
            {
                using var frame = sender.TryGetNextFrame();
                Console.WriteLine($"frame arrived: {frame.ContentSize.Width}x{frame.ContentSize.Height}");
            };
            var session = framePool.CreateCaptureSession(captureItem);
            session.StartCapture();
        }
    }
}