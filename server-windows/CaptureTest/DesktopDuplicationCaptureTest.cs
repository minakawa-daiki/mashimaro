using System.Drawing;
using System.IO;
using System.Linq;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using Capture;
using Media.Rtsp.Server.MediaTypes;
using Xunit;

namespace CaptureTest
{
    public class DesktopDuplicationCaptureTest
    {
        [Fact]
        public void TestCaptureJpeg()
        {
            const int quality = 80;
            
            var capture = new DesktopDuplicationCapture(500);
            var encoder = new JpegBitmapEncoder();
            encoder.QualityLevel = quality;
            var captured = false;
            capture.FrameArrived += (sender, f) =>
            {
                var bitmapSource = BitmapSource.Create(f.Size.Width, f.Size.Height, 96, 96, PixelFormats.Bgr32, null,
                    f.Buffer, f.RowPitch * f.Size.Height, f.RowPitch);
                encoder.Frames.Add(BitmapFrame.Create(bitmapSource));
                captured = true;
            };
            
            var captureRegion = new Rectangle(0, 0, 500, 500);
            while (!captured) capture.Capture(captureRegion);

            using var stream = new MemoryStream();
            encoder.Save(stream);

            var rtpf = new RFC2435Media.RFC2435Frame(stream, quality);
            var packets = rtpf.ToArray();
            Image j = rtpf;
            j.Save("rtpsave.jpg");
        }
    }
}