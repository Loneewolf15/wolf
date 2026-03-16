<?php
class Shops extends Controller
{


    public function __construct()
    {
        $this->shopModel = $this->model('Shop');
        $this->userModel = $this->model('User');
        $this->serverKey  = 'secret_server_key';
    }

    public function createShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();

            $data = [
                'shop_name' => ($loginData['shop_name']),
                'shop_desc' => ($loginData['shop_desc']),
                'shop_whatsapp_link' => ($loginData['shop_whatsapp_link']),
                'service_offered' => ($loginData['service_offered']),
                'shop_image_url' => ($loginData['shop_image_url']),
                'university' => ($loginData['university']),
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'shop_name_err' => '',
                'shop_desc_err' => '',
                'shop_whatsapp_link_err' => '',
                'service_offered_err' => '',
                'university_err' => '',
                'shop_image_url_err' => ''
            ];
            // print_r(json_encode($data));
            // exit;


            // Validate shop name
            if (empty($data['shop_name'])) {
                $data['shop_name_err'] = 'Please enter the shop name';
            }
            $data['shop_tag'] = $this->generateShopTag($data['shop_name']);
            // Validate shop description
            if (empty($data['shop_desc'])) {
                $data['shop_desc_err'] = 'Please enter the shop description';
            }
            // Validate shop description
            if (empty($data['university'])) {
                $data['university_err'] = 'Please enter the shop description';
            }

            // Validate shop WhatsApp link
            if (empty($data['shop_whatsapp_link'])) {
                $data['shop_whatsapp_link_err'] = 'Please enter the shop WhatsApp link';
            }

            // Validate service offered
            if (empty($data['service_offered'])) {
                $data['service_offered_err'] = 'Please enter the services offered';
            }

            // Validate shop image URL
            if (empty($data['shop_image_url'])) {
                $data['shop_image_url_err'] = 'Please enter the shop image URL';
            }

            // Make sure there are no errors
            if (
                empty($data['shop_name_err']) &&
                empty($data['shop_desc_err']) &&
                empty($data['shop_whatsapp_link_err']) &&
                empty($data['shop_image_url_err']) &&
                empty($data['university_err'])
                
            ) {
                // Validation passed
                //Execute
                if ($this->shopModel->createShop($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'Shop created',
                        'userData' => $userData
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to create shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    
    
    public function generateShopTag($shopName) {
    // Convert the shop name to lowercase
    $shopTag = strtolower($shopName);
    
    // Remove apostrophes
    $shopTag = str_replace("'", "", $shopTag);
    
    // Remove spaces
    $shopTag = str_replace(" ", "", $shopTag);
    
    return $shopTag;
}

    
    
    public function editShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $user_id = $userData->user_id;

            $data = [
                'shop_name' => ($loginData['shop_name']),
                'shop_desc' => ($loginData['shop_desc']),
                'shop_whatsapp_link' => ($loginData['shop_whatsapp_link']),
                'service_offered' => ($loginData['service_offered']),
                'shop_image_url' => ($loginData['shop_image_url']),
                'shop_location' => ($loginData['shop_location']),
                'seller_id' => $userData->user_id,
                'shop_id' => $userData->shop_id,
                'shop_name_err' => '',
                'shop_desc_err' => '',
                'shop_whatsapp_link_err' => '',
                'service_offered_err' => '',
                'shop_image_url_err' => ''
            ];
            
            // print_r(json_encode($data));
            // exit;


            // Validate shop name
            if (empty($data['shop_name'])) {
                $data['shop_name_err'] = 'Please enter the shop name';
            }
             $data['shop_tag'] = $this->generateShopTag($data['shop_name']);
            // Validate shop description
            if (empty($data['shop_desc'])) {
                $data['shop_desc_err'] = 'Please enter the shop description';
            }

            // Validate shop WhatsApp link
            if (empty($data['shop_whatsapp_link'])) {
                $data['shop_whatsapp_link_err'] = 'Please enter the shop WhatsApp link';
            }

            // Validate service offered
            if (empty($data['service_offered'])) {
                $data['service_offered_err'] = 'Please enter the services offered';
            }

            // Validate shop image URL
            if (empty($data['shop_image_url'])) {
                $data['shop_image_url_err'] = 'Please enter the shop image URL';
            }

            // Make sure there are no errors
            if (
                empty($data['shop_name_err']) &&
                empty($data['shop_desc_err']) &&
                empty($data['shop_whatsapp_link_err']) &&
                empty($data['shop_image_url_err'])
            ) {
                // /Validation passed
                //Execute
                if ($this->shopModel->createShop22($data)) {
                    $response = [
                        'status' => true,
                        'message' => 'Shop updated',
                        'userData' => $userData,
                         'shopDetails' => $this->shopModel->getShop($user_id, $userData->shop_id),
                         
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to create shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    public function getShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        $shopDetails = $this->shopModel->getShop($userData->user_id);
        print_r(json_encode($shopDetails));
      
    }
    public function getShopByUni()
    {
      $data = $this->getData();
      $uni = $data['university'];
        $shopDetails = $this->shopModel->getShopByUni($uni);
        print_r(json_encode($shopDetails));
      
    }
    public function getUni()
    {
        $shopDetails = $this->shopModel->getUni();
        print_r(json_encode($shopDetails));
      
    }
    public function getAllShops()
    {
        // try {
        //     $userData = $this->RouteProtection();
        // } catch (UnexpectedValueException $e) {
        //     $res = [
        //         'status' => 401,
        //         'message' =>  $e->getMessage(),
        //     ];
        //     print_r(json_encode($res));
        //     exit;
        // }
        $shopDetails = $this->shopModel->getAllShops();
        print_r(json_encode($shopDetails));
      
    }
    public function activateShop()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $shopID = "shop".md5($str);
            $data = [
                'shop_plan' => ($loginData['shop_plan']),
                'plan_price' => ($loginData['plan_price']),
                'expiry_date' => ($loginData['expiry_date']),
                'seller_id' => $userData->user_id,
                'shop_id' => $shopID,
                'shop_plan_err' => '',
                'plan_price_err' => '',
                'expiry_date_err' => ''
            ];


            // Validate shop name
            if (empty($data['shop_plan'])) {
                $data['shop_plan_err'] = 'Please enter the shop name';
            }

            // Validate shop description
            if (empty($data['plan_price'])) {
                $data['plan_price_err'] = 'Please enter the shop description';
            }

            // Validate shop WhatsApp link
            if (empty($data['expiry_date'])) {
                $data['expiry_date_err'] = 'Please enter the shop WhatsApp link';
            }


            // Make sure there are no errors
            if (
                empty($data['shop_plan_err']) &&
                empty($data['plan_price_err']) &&
                empty($data['expiry_date_err'])
            ) {
                // Validation passed
                //Execute
                if ($this->shopModel->activateShop($data)) {
                    $this->Shopemailer($userData->email, "Welcome to vPlaza Store");
                    $response = [
                        'status' => true,
                        'message' => 'Shop activated'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to activate shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
           
        }
    
    }
    public function activateShop2()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $shopID = "shop".md5($str);
            $data = [
                'shop_plan' => "1 month",
                'plan_price' => "free",
                'expiry_date' => "30 days time",
                'seller_id' => $userData->user_id,
                'shop_id' => $shopID,
                'shop_plan_err' => '',
                'plan_price_err' => '',
                'expiry_date_err' => ''
            ];


            // Validate shop name
            if (empty($data['shop_plan'])) {
                $data['shop_plan_err'] = 'Please enter the shop name';
            }

            // Validate shop description
            if (empty($data['plan_price'])) {
                $data['plan_price_err'] = 'Please enter the shop description';
            }

            // Validate shop WhatsApp link
            if (empty($data['expiry_date'])) {
                $data['expiry_date_err'] = 'Please enter the shop WhatsApp link';
            }


            // Make sure there are no errors
            if (
                empty($data['shop_plan_err']) &&
                empty($data['plan_price_err']) &&
                empty($data['expiry_date_err'])
            ) {
                // Validation passed
                //Execute
                if ($this->shopModel->activateShop($data)) {
                    $this->Shopemailer($userData->email, "Welcome to vPlaza Store");
                    $response = [
                        'status' => true,
                        'message' => 'Shop activated'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to activate shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
            } else {
                $response = [
                    'status' => false,
                    'message' => 'missing param',
                    'data' => $data
               ];
               print_r(json_encode($response));
               exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }
    
    
    public function productRequest()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' =>  $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $loginData = $this->getData();
            $str = date('ymdhms');
            $shopID = "req".md5($str);
            $data = [
                'product_name' => $loginData['product_name'],
                'price' => $loginData['price'],
                'number' => $loginData['number'],
                'img' => $loginData['img'],
                'desc' => $loginData['desc'],
                'location' => $loginData['location'],
                'request_id' => $shopID
            ];


            // Validate shop name
            if (empty($data['product_name'])) {
                 $response = [
                        'status' => false,
                        'message' => 'enter product name'
                   ];
                   print_r(json_encode($response));
                   exit;
            }
            // Validate shop name
            if (empty($data['location'])) {
                 $response = [
                        'status' => false,
                        'message' => 'enter your location'
                   ];
                   print_r(json_encode($response));
                   exit;
            }


            // Validate shop description
            if (empty($data['number'])) {
                  $response = [
                        'status' => false,
                        'message' => 'enter your whatsapp number'
                   ];
                   print_r(json_encode($response));
                   exit;
            }

            // Validate shop WhatsApp link
            if (empty($data['img'])) {
                  $response = [
                        'status' => false,
                        'message' => 'enter image url'
                   ];
                   print_r(json_encode($response));
                   exit;
            }

$shopDetails = $this->shopModel->getAllShop($data['location']);
                if ($this->shopModel->saveRequest($data)) {
                    
                    foreach ($shopDetails as $item) {
                        $userDetails = $this->userModel->getUser($item->seller_id);
                         $this->loopMailer($userDetails->email, "Product Request...");
                    }
                   
                    $response = [
                        'status' => true,
                        'message' => 'product Requeted'
                   ];
                   print_r(json_encode($response));
                   exit;
                } else {
                   $response = [
                        'status' => false,
                        'message' => 'failed to activate shop'
                   ];
                   print_r(json_encode($response));
                   exit;
                }
           
        } else {
            $response = [
                'status' => false,
                'message' => 'wrong method'
           ];
           print_r(json_encode($response));
           exit;
        }
    }

    // Edit Post
    public function edit($id)
    {
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            // Sanitize POST
            $_POST  = filter_input_array(INPUT_POST, FILTER_SANITIZE_STRING);

            $data = [
                'id' => $id,
                'title' => ($_POST['title']),
                'body' => ($_POST['body']),
                'user_id' => $_SESSION['user_id'],
                'title_err' => '',
                'body_err' => ''
            ];

            // Validate email
            if (empty($data['title'])) {
                $data['title_err'] = 'Please enter name';
                // Validate name
                if (empty($data['body'])) {
                    $data['body_err'] = 'Please enter the post body';
                }
            }

            // Make sure there are no errors
            if (empty($data['title_err']) && empty($data['body_err'])) {
                // Validation passed
                //Execute
                if ($this->postModel->updatePost($data)) {
                    // Redirect to login
                    flash('post_message', 'Post Updated');
                    redirect('posts');
                } else {
                    die('Something went wrong');
                }
            } else {
                // Load view with errors
                $this->view('posts/edit', $data);
            }
        } else {
            // Get post from model
            $post = $this->postModel->getPostById($id);

            // Check for owner
            if ($post->user_id != $_SESSION['user_id']) {
                redirect('posts');
            }

            $data = [
                'id' => $id,
                'title' => $post->title,
                'body' => $post->body,
            ];

            $this->view('posts/edit', $data);
        }
    }

    // Delete Post
    public function delete($id)
    {
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            //Execute
            if ($this->postModel->deletePost($id)) {
                // Redirect to login
                flash('post_message', 'Post Removed');
                redirect('posts');
            } else {
                die('Something went wrong');
            }
        } else {
            redirect('posts');
        }
    }
}
