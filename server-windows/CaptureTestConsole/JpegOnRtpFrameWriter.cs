using System;
using System.IO;
using System.Linq;
using System.Net.Sockets;
using System.Threading.Tasks;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using Media.Rtsp.Server.MediaTypes;

namespace CaptureTestConsole
{
    public class JpegOnRtpFrameWriter : IFrameWriter
    {
        private readonly UdpClient _udpClient;
        private readonly int _quality;
        private ushort _seqNo;
        private readonly object _mutex = new object();
        private readonly int _mtu;
        
        public JpegOnRtpFrameWriter(UdpClient udpClient, int quality, int mtu)
        {
            _udpClient = udpClient;
            _quality = quality;
            _mtu = mtu;
        }
        public void WriteFrame(int width, int height, int rowPitch, IntPtr buffer)
        {
            lock (_mutex)
            {
                var bitmapSource = BitmapSource.Create(width, height, 96, 96, PixelFormats.Bgr32, null,
                    buffer, rowPitch * height, rowPitch);
                var encoder = new JpegBitmapEncoder();
                encoder.QualityLevel = _quality;
                encoder.Frames.Add(BitmapFrame.Create(bitmapSource));
                
                using var stream = new MemoryStream();
                encoder.Save(stream);
                var rtpFrames = new RFC2435Media.RFC2435Frame(stream, encoder.QualityLevel, bytesPerPacket: _mtu - 12);
                var tasks = rtpFrames.Select(packet =>
                {
                    _seqNo++;
                    var timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
                    var bytes = packet.Prepare(null, null, timestamp: (int) timestamp, sequenceNumber: _seqNo).ToArray();
                    return _udpClient.SendAsync(bytes, bytes.Length);
                });
                Task.WhenAll(tasks);
            }
        }
    }
}