import { TemplateLocale } from '../schemas';

export interface TemplateSeed {
  name: string;
  locale: TemplateLocale;
  subject: string;
  body: string;
  description: string;
}

export const defaultTemplates: TemplateSeed[] = [
  // E-Ticket - Thai
  {
    name: 'e_ticket',
    locale: TemplateLocale.TH,
    subject: '🎫 E-Ticket ของคุณพร้อมแล้ว - {{event_name}}',
    body: `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
    .header h1 { margin: 0; font-size: 24px; }
    .content { padding: 30px; }
    .ticket-box { background: #f8f9fa; border-radius: 8px; padding: 20px; margin: 20px 0; border-left: 4px solid #667eea; }
    .qr-code { text-align: center; margin: 30px 0; }
    .qr-code img { max-width: 200px; }
    .info-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #eee; }
    .info-label { color: #666; }
    .info-value { font-weight: bold; color: #333; }
    .confirmation-code { font-size: 24px; font-weight: bold; color: #667eea; text-align: center; letter-spacing: 2px; }
    .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>🎫 E-Ticket</h1>
      <p>การจองของคุณเสร็จสมบูรณ์แล้ว!</p>
    </div>
    <div class="content">
      <h2>{{event_name}}</h2>

      <div class="ticket-box">
        <div class="info-row">
          <span class="info-label">วันที่</span>
          <span class="info-value">{{show_date}}</span>
        </div>
        <div class="info-row">
          <span class="info-label">โซน</span>
          <span class="info-value">{{zone_name}}</span>
        </div>
        <div class="info-row">
          <span class="info-label">จำนวน</span>
          <span class="info-value">{{quantity}} ที่นั่ง</span>
        </div>
        <div class="info-row">
          <span class="info-label">สถานที่</span>
          <span class="info-value">{{venue_name}}</span>
        </div>
      </div>

      <div class="qr-code">
        <p>แสดง QR Code นี้ที่ทางเข้างาน</p>
        <img src="{{qr_code_url}}" alt="QR Code">
      </div>

      <p style="text-align: center;">รหัสยืนยัน</p>
      <p class="confirmation-code">{{confirmation_code}}</p>
    </div>
    <div class="footer">
      <p>Booking Rush - High-Performance Ticket Booking</p>
      <p>หากมีคำถาม ติดต่อ support@bookingrush.com</p>
    </div>
  </div>
</body>
</html>
    `,
    description: 'E-Ticket template with QR code - Thai',
  },

  // E-Ticket - English
  {
    name: 'e_ticket',
    locale: TemplateLocale.EN,
    subject: '🎫 Your E-Ticket is Ready - {{event_name}}',
    body: `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
    .header h1 { margin: 0; font-size: 24px; }
    .content { padding: 30px; }
    .ticket-box { background: #f8f9fa; border-radius: 8px; padding: 20px; margin: 20px 0; border-left: 4px solid #667eea; }
    .qr-code { text-align: center; margin: 30px 0; }
    .qr-code img { max-width: 200px; }
    .info-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #eee; }
    .info-label { color: #666; }
    .info-value { font-weight: bold; color: #333; }
    .confirmation-code { font-size: 24px; font-weight: bold; color: #667eea; text-align: center; letter-spacing: 2px; }
    .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>🎫 E-Ticket</h1>
      <p>Your booking is confirmed!</p>
    </div>
    <div class="content">
      <h2>{{event_name}}</h2>

      <div class="ticket-box">
        <div class="info-row">
          <span class="info-label">Date</span>
          <span class="info-value">{{show_date}}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Zone</span>
          <span class="info-value">{{zone_name}}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Quantity</span>
          <span class="info-value">{{quantity}} seat(s)</span>
        </div>
        <div class="info-row">
          <span class="info-label">Venue</span>
          <span class="info-value">{{venue_name}}</span>
        </div>
      </div>

      <div class="qr-code">
        <p>Show this QR Code at the entrance</p>
        <img src="{{qr_code_url}}" alt="QR Code">
      </div>

      <p style="text-align: center;">Confirmation Code</p>
      <p class="confirmation-code">{{confirmation_code}}</p>
    </div>
    <div class="footer">
      <p>Booking Rush - High-Performance Ticket Booking</p>
      <p>Questions? Contact support@bookingrush.com</p>
    </div>
  </div>
</body>
</html>
    `,
    description: 'E-Ticket template with QR code - English',
  },

  // Payment Receipt - Thai
  {
    name: 'payment_receipt',
    locale: TemplateLocale.TH,
    subject: '✅ ชำระเงินสำเร็จ - {{event_name}}',
    body: `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: #22c55e; color: white; padding: 30px; text-align: center; }
    .content { padding: 30px; }
    .receipt-box { background: #f8f9fa; border-radius: 8px; padding: 20px; margin: 20px 0; }
    .total { font-size: 28px; font-weight: bold; color: #22c55e; text-align: center; margin: 20px 0; }
    .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>✅ ชำระเงินสำเร็จ</h1>
    </div>
    <div class="content">
      <p>ขอบคุณสำหรับการชำระเงิน</p>

      <div class="receipt-box">
        <p><strong>Event:</strong> {{event_name}}</p>
        <p><strong>รหัสการจอง:</strong> {{confirmation_code}}</p>
        <p><strong>รหัสการชำระเงิน:</strong> {{payment_id}}</p>
        <p><strong>วิธีการชำระเงิน:</strong> {{payment_method}}</p>
        <p><strong>จำนวน:</strong> {{quantity}} ที่นั่ง</p>
      </div>

      <p class="total">฿{{total_price}}</p>

      <p style="text-align: center; color: #666;">E-Ticket จะถูกส่งในอีเมลแยกต่างหาก</p>
    </div>
    <div class="footer">
      <p>Booking Rush - High-Performance Ticket Booking</p>
    </div>
  </div>
</body>
</html>
    `,
    description: 'Payment receipt template - Thai',
  },

  // Booking Expired - Thai
  {
    name: 'booking_expired',
    locale: TemplateLocale.TH,
    subject: '⏰ การจองของคุณหมดอายุแล้ว - {{event_name}}',
    body: `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: #f59e0b; color: white; padding: 30px; text-align: center; }
    .content { padding: 30px; }
    .btn { display: inline-block; background: #667eea; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; margin-top: 20px; }
    .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>⏰ การจองหมดอายุ</h1>
    </div>
    <div class="content">
      <p>การจองของคุณสำหรับ <strong>{{event_name}}</strong> หมดอายุแล้ว เนื่องจากไม่ได้ชำระเงินภายในเวลาที่กำหนด</p>

      <p>หากยังต้องการซื้อตั๋ว กรุณาจองใหม่อีกครั้ง (ขึ้นอยู่กับความพร้อมของที่นั่ง)</p>

      <p style="text-align: center;">
        <a href="{{rebook_url}}" class="btn">จองใหม่</a>
      </p>
    </div>
    <div class="footer">
      <p>Booking Rush - High-Performance Ticket Booking</p>
    </div>
  </div>
</body>
</html>
    `,
    description: 'Booking expired notification - Thai',
  },

  // Booking Cancelled - Thai
  {
    name: 'booking_cancelled',
    locale: TemplateLocale.TH,
    subject: '❌ การจองถูกยกเลิก - {{event_name}}',
    body: `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
    .header { background: #ef4444; color: white; padding: 30px; text-align: center; }
    .content { padding: 30px; }
    .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>❌ การจองถูกยกเลิก</h1>
    </div>
    <div class="content">
      <p>การจองของคุณสำหรับ <strong>{{event_name}}</strong> ถูกยกเลิกแล้ว</p>

      <p><strong>รหัสการจอง:</strong> {{confirmation_code}}</p>

      <p>หากคุณชำระเงินแล้ว จะได้รับเงินคืนภายใน 3-5 วันทำการ</p>
    </div>
    <div class="footer">
      <p>Booking Rush - High-Performance Ticket Booking</p>
    </div>
  </div>
</body>
</html>
    `,
    description: 'Booking cancelled notification - Thai',
  },
];
