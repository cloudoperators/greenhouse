---
title: "SAP ID Service"
weight: 3
---

This section provides a step-by-step walkthrough for new users to request an SAP ID Service (IDS) tenant.

## How to request an SAP ID Service (IDS) tenant

### Prerequisites

* Cost Center
* Global Account ID
    * You can find the Global Account ID in the SAP BTP Cockpit under **Sub account Overview** > **Account ID**.
    * If you don't have a Global Account yet, please create one in the [SAP BTP Control Center](https://controlcenter.ondemand.com/)

### Requesting an IDS tenant

1. Open the [SCI Tenant Registration](https://tenants.ias.only.sap/)
2. Click **Request a new tenant**.
3. Select **Internal Tenant** as the **Tenant Type**.
4. Enter the **Cost Center**.
5. Select the Landscape where you want to create the tenant.
6. Select the Tenant type.
7. Enter the **Global Account ID**.
8. Click **Submit**.

## Requesting an Azure Enterprise Application

1. Open the [SAP IT Service Portal](https://itsupportportal.services.sap/itsupportlegacy?id=itsm_sc_cat_item&table=sc_cat_item&sys_id=ab5d0a6b1b8ab0105039a6c8bb4bcba7).
2. Enter **Application Owners** (at least two required)
3. Enter the **Application Name** (need to start with *SAP SE:*)
4. Accept the **Terms and Conditions**.
5. Click **Submit**.

## Configuring the SAP ID Service (IDS) tenant

1. Follow the steps in the email you received from **SAP ID Service** with subject: `Activation Information for SAP Cloud Identity Services` to activate your account and set your password.
2. Open the Administration Console of your tenant. *(e.g. https://<tenant-name>.accounts.ondemand.com/admin)*
3. Click **Tenant Settings** under **Applications & Resources**
4. Click **SAML 2.0 Configuration** in **Signle Sign-on** section.
5. Click on **Download Metadata File** and save the file.
6. Turn on **IdP-Initiated SSO** in **Signle Sign-on** section.
7. Click on **Corporate Identity Providers** under **Identity Providers**.
8. Click on **Create**.
9. Enter the **Name** (e.g. the name of your Azure Enterprise Application).
10. Select **Microsoft ADFS / Azure AD (SAML 2.0)** as the **Type**.
11. Click **Save**.
12. Click on **SAML 2.0 Configuration** in your newly created Provider.
13. Enter **Metadata URL**: `https://login.microsoftonline.com/42f7676c-f455-423c-82f6-dc2d99791af7/federationmetadata/2007-06/federationmetadata.xml` and click **Load**.
14. Click **Save**.
15. Turn on **Forward All SSO Requests to Corporate IdP** in **Signle Sign-on** section.
16. Click on **Identity Federation** and turn on **Use Identity Authentication user store**.
17. Next step is to configure CAM to automatically create users in the SAP ID Service (IDS) tenant. This action requires a **CAM Admin** to be performed.
    Please open the [Cloud LoB Areas and Responsibilities](https://spc.ondemand.com/sap/bc/webdynpro/a1sspc/cam_wd_central?item=areas) report to see all available CAM areas and its admins.
18. Your CAM admin needs to follow the steps in the [IAS - CAM configuration step-by-step guide](https://wiki.one.int.sap/wiki/display/CLMAM/IAS+-+CAM+configuration+step-by-step+guide)
19. Under **Applications & Resources** click on **Administration Console** and **Conditional Authentication**.
20. Change the **Default Authenticating Identity Provider** from **Identity Authentication** to the one you created.
21. Click **Save**.
22. Repeat step 20-21 for **User Profile** application.

## Configuring the Azure Enterprise Application

1. Open the [Azure Portal](https://portal.azure.com/#view/Microsoft_AAD_IAM/StartboardApplicationsMenuBlade/~/AppAppsPreview). If you don't have access to Azure Portal you can request it [here](https://myaccess.microsoft.com/@sap.onmicrosoft.com#/access-packages/d55ad7db-69af-4da5-8520-de187364513d)
2. Search for the **Enterprise Application** by its name.
3. Click **Properties**.
4. Click **No** for **User assignment required?**.
5. Click **Save**.
6. Click **Single sign-on**.
7. Click **SAML**.
8. Click **Upload metadata file**.
9. Select the **metadata.xml** previously downloaded from **IDS**
10. Click **Add**.
11. Click **Save** on the pop-up window
12. Download the **Federation Metadata XML** from the **(3) SAML Certificates** section.
13. Click on **Edit** in **(2) Attributes & Claims**. The Clams Mapping window opens. The Claim mapping should look like this:

> Required claims

| Claim name   |      Type      |  Value |
|----------|:-------------:|------:|
| Unique User Identifier (Name ID) |  SAML | user.mail [nameid-format:emailAddress] |

> Additional claims

| Claim name   |      Type      |  Value |
|----------|:-------------:|------:|
| company |  SAML | user.companyname |
| email |    SAML   |   user.mail |
| http://schemas.microsoft.com/ws/2008/06/identity/claims/groups | SAML | user.groups [SecurityGroup] |
| http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress | SAML | user.mail |
| http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname | SAML | user.givenname |
| http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name | SAML | user.userprincipalname |
| http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname | SAML | user.surname |
| uid | SAML | user.mailnickname |
| user_uuid | SAML | user.objectid |

14. Navigate to [**App Registrations**](https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade) in Azure Portal.
15. Click on your **Application**.
16. Click on **API permissions** then on **Add a permission**.
17. Click on **Microsoft Graph** then **Delegated permissions**.
18. Select **User.Read** and **User.ReadBasic.All** under **User** Permissions then click **Add permissions**.
17. Your **Enterprise Application** is now ready to be used with **IDS**.