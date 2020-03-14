using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.Runtime.InteropServices;
using Capture;
using WinApi.Gdi32;
using Xunit;
using Bitmap = System.Drawing.Bitmap;

namespace CaptureTest
{
    public class DesktopDuplicationCaptureTest
    {
        [Fact]
        public void TestCapture()
        {
            var capture = new DesktopDuplicationCapture();
            byte[] frameBytes = null;
            capture.FrameArrived += (sender, f) => { frameBytes = f.GetBytes(); };
            var captureRegion = new Rectangle(0, 0, 500, 500);
            while (frameBytes == null) capture.Capture(captureRegion);
            Assert.NotNull(frameBytes);
            using var frameBytesStream = new MemoryStream(frameBytes);
            using var bitmap = new Bitmap(captureRegion.Width, captureRegion.Height, PixelFormat.Format32bppArgb);
            var bitmapData = bitmap.LockBits(new Rectangle(0, 0, bitmap.Width, bitmap.Height), ImageLockMode.WriteOnly, bitmap.PixelFormat);
            Marshal.Copy(frameBytes, 0, bitmapData.Scan0, frameBytes.Length);
            bitmap.UnlockBits(bitmapData);
            bitmap.Save("capture.bmp");
            Assert.Equal(captureRegion.Width, bitmap.Width);
            Assert.Equal(captureRegion.Height, bitmap.Height);
        }
    }
}